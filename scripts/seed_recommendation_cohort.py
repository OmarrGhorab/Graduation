"""
Seed a multi-user cohort with realistic learning analytics so the
recommendation service's clustering / collaborative-filtering signal
(clusterContribution) becomes non-zero.

It does NOT recreate courses/subjects — it reuses whatever catalog already
exists and layers students + enrollments + user_course_analytics on top.

Cohort design: students are split into subject personas. Within a persona,
members enroll in an overlapping pool of courses but each leaves "gaps"
(courses their persona-mates took that they did not). Those gaps are exactly
what the cluster boost should surface as recommendations.

The existing demo student (student@example.com) is kept and placed in the
Mobile/UX persona, so logging in as that user shows the boost end-to-end.
"""
import os
import uuid
import random

import bcrypt
import psycopg2
from psycopg2.extras import execute_values
from datetime import datetime, timedelta

DB_URL = os.environ.get("SEED_DB_URL", "postgresql://graduation:graduation_secret@localhost:5432/graduation")
DEMO_STUDENT_EMAIL = "student@example.com"

random.seed(42)  # reproducible cohort


def get_password_hash(password: str) -> str:
    return bcrypt.hashpw(password.encode("utf-8"), bcrypt.gensalt()).decode("utf-8")


# Personas: which subjects a group of learners cares about. Each persona gets
# its own set of students; the demo student joins "mobile_ux".
PERSONAS = {
    "mobile_ux": {
        "subjects": ["Mobile Development", "UI/UX Design"],
        "students": [
            ("maya.dev@example.com", "Maya Mobile"),
            ("liam.ui@example.com", "Liam UX"),
            ("noor.app@example.com", "Noor AppDev"),
            ("omar.front@example.com", "Omar Frontend"),
        ],
    },
    "data_science": {
        "subjects": ["Data Science"],
        "students": [
            ("sara.data@example.com", "Sara Data"),
            ("youssef.ml@example.com", "Youssef ML"),
            ("hana.stats@example.com", "Hana Stats"),
            ("karim.ai@example.com", "Karim AI"),
        ],
    },
    "cloud": {
        "subjects": ["Cloud Architecture"],
        "students": [
            ("adel.cloud@example.com", "Adel Cloud"),
            ("rana.devops@example.com", "Rana DevOps"),
            ("tarek.infra@example.com", "Tarek Infra"),
        ],
    },
}


def seed():
    conn = psycopg2.connect(DB_URL)
    cur = conn.cursor()
    now = datetime.utcnow()
    try:
        # --- catalog -------------------------------------------------------
        cur.execute("SELECT id, name FROM public.subjects")
        subjects = {name: sid for sid, name in cur.fetchall()}
        if not subjects:
            raise RuntimeError("No subjects found — run reseed_all.py first")

        cur.execute("SELECT id, subject_id, total_lessons FROM public.courses")
        courses_by_subject = {}
        for cid, subject_id, total_lessons in cur.fetchall():
            courses_by_subject.setdefault(subject_id, []).append((cid, total_lessons or 5))
        print(f"Catalog: {len(subjects)} subjects, "
              f"{sum(len(v) for v in courses_by_subject.values())} courses")

        cur.execute("SELECT id FROM auth.\"User\" WHERE role='TEACHER' LIMIT 1")
        row = cur.fetchone()
        teacher_id = row[0] if row else None

        # Interests must exist in auth."Interest" (id mirrors subject id in seeders)
        cur.execute('SELECT id, name FROM auth."Interest"')
        interest_ids = {name: iid for iid, name in cur.fetchall()}

        def ensure_interest(name):
            if name in interest_ids:
                return interest_ids[name]
            iid = subjects.get(name, str(uuid.uuid4()))
            cur.execute(
                'INSERT INTO auth."Interest" (id, name, "createdAt", "updatedAt") '
                'VALUES (%s,%s,%s,%s) ON CONFLICT (id) DO NOTHING',
                (iid, name, now, now),
            )
            interest_ids[name] = iid
            return iid

        # --- clean prior cohort analytics so re-runs are idempotent --------
        cohort_emails = [e for p in PERSONAS.values() for e, _ in p["students"]]
        cur.execute('SELECT id, email FROM auth."User" WHERE email = ANY(%s)', (cohort_emails,))
        existing = {email: uid for uid, email in cur.fetchall()}

        pwd = get_password_hash("password123")

        def get_or_create_student(email, name, interests):
            if email in existing:
                uid = existing[email]
            else:
                uid = str(uuid.uuid4())
                cur.execute(
                    'INSERT INTO auth."User" (id, name, username, email, password, role, '
                    '"onboardingCompleted", verified, "createdAt", "updatedAt") '
                    "VALUES (%s,%s,%s,%s,%s,'STUDENT',true,true,%s,%s)",
                    (uid, name, email.split("@")[0], email, pwd, now, now),
                )
                existing[email] = uid
            # (re)assign interests
            cur.execute('DELETE FROM auth."UserInterest" WHERE "userId"=%s', (uid,))
            rows = [(uid, ensure_interest(i), now) for i in interests]
            if rows:
                execute_values(
                    cur,
                    'INSERT INTO auth."UserInterest" ("userId","interestId","createdAt") VALUES %s '
                    "ON CONFLICT DO NOTHING",
                    rows,
                )
            return uid

        # demo student joins the mobile_ux persona
        cur.execute('SELECT id FROM auth."User" WHERE email=%s', (DEMO_STUDENT_EMAIL,))
        demo_row = cur.fetchone()
        demo_id = demo_row[0] if demo_row else None

        def analytics_row(user_id, course_id, total_lessons, strong):
            completed = total_lessons if strong else max(1, total_lessons // 3)
            completion = round(100.0 * completed / max(total_lessons, 1), 2)
            avg_watch = round(random.uniform(70, 98) if strong else random.uniform(25, 55), 2)
            engagement = round(random.uniform(70, 95) if strong else random.uniform(30, 55), 2)
            watch_time = int((random.uniform(180, 360)) * completed)
            return (
                str(uuid.uuid4()), course_id, user_id, watch_time,
                total_lessons, completed, total_lessons, completion,
                avg_watch, engagement, now, now, now,
            )

        all_cohort_ids = []
        analytics_values = []
        enrollment_values = []

        for persona_key, persona in PERSONAS.items():
            subj_names = persona["subjects"]
            pool = []
            for sn in subj_names:
                sid = subjects.get(sn)
                if sid:
                    pool.extend(courses_by_subject.get(sid, []))
            if not pool:
                print(f"  ! persona {persona_key}: no courses for {subj_names}, skipping")
                continue

            members = [get_or_create_student(e, n, subj_names) for e, n in persona["students"]]
            if persona_key == "mobile_ux" and demo_id:
                members.append(demo_id)

            print(f"  persona {persona_key}: {len(members)} students, {len(pool)} courses in pool")

            for mi, uid in enumerate(members):
                all_cohort_ids.append(uid)
                # Each member takes most of the pool but skips a rotating slice,
                # guaranteeing peers have courses this member lacks (the gap = boost target).
                skip = {pool[(mi + k) % len(pool)][0] for k in range(min(2, len(pool) - 1))}
                # The demo student deliberately takes the *least* so it has the most to be recommended.
                take_strong = mi % 2 == 0
                for cid, total_lessons in pool:
                    if uid == demo_id and cid in skip:
                        continue
                    if cid in skip:
                        continue
                    enrollment_values.append((str(uuid.uuid4()), uid, cid, True, now, now))
                    analytics_values.append(analytics_row(uid, cid, total_lessons, take_strong))

        # --- wipe & insert analytics/enrollments for the cohort ------------
        if all_cohort_ids:
            cur.execute("DELETE FROM public.user_course_analytics WHERE user_id = ANY(%s::uuid[])", (all_cohort_ids,))
            cur.execute("DELETE FROM public.enrollments WHERE user_id = ANY(%s::uuid[])", (all_cohort_ids,))

        execute_values(
            cur,
            "INSERT INTO public.enrollments (id, user_id, course_id, is_active, enrolled_at, updated_at) "
            "VALUES %s ON CONFLICT (course_id, user_id) DO NOTHING",
            enrollment_values,
        )
        execute_values(
            cur,
            "INSERT INTO public.user_course_analytics "
            "(id, course_id, user_id, total_watch_time, lessons_started, lessons_completed, "
            " total_lessons, completion_pct, avg_lesson_watch_pct, engagement_score, "
            " last_activity_at, created_at, updated_at) VALUES %s "
            "ON CONFLICT (user_id, course_id) DO NOTHING",
            analytics_values,
        )

        conn.commit()
        print(f"\nSeeded {len(all_cohort_ids)} cohort users, "
              f"{len(enrollment_values)} enrollments, {len(analytics_values)} analytics rows.")
        print(f"Demo student ({DEMO_STUDENT_EMAIL}) id={demo_id} placed in mobile_ux persona.")
        print("All cohort passwords: password123")
    except Exception as exc:
        conn.rollback()
        print(f"ERROR: {exc}")
        raise
    finally:
        cur.close()
        conn.close()


if __name__ == "__main__":
    seed()
