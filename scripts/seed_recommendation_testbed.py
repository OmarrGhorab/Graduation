"""
Reset and seed a coherent recommendation testbed.

Creates:
- 8 subjects/interests
- 1 teacher account
- 10 student accounts with clear personas
- 100 courses with titles that match their subjects
- lessons, enrollments, watch events, lesson progress, and course analytics

Run from repo root:
    python scripts/seed_recommendation_testbed.py

Optional:
    SEED_DB_URL=postgresql://... python scripts/seed_recommendation_testbed.py
"""

from __future__ import annotations

import os
import random
import uuid
from datetime import datetime, timedelta, timezone
from typing import Dict, Iterable, List, Sequence, Tuple

import psycopg2
from psycopg2.extras import execute_values


DB_URL = os.environ.get(
    "SEED_DB_URL",
    "postgresql://graduation:graduation_secret@localhost:5432/graduation",
)

random.seed(20260607)
PASSWORD123_BCRYPT = "$2b$10$g8KarQiNmMLuYdMOoNG4wum/2dUj9H7EgVTACpvwwoB55JAtUgPfC"

THUMBS = [
    "https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=800&q=80",
    "https://images.unsplash.com/photo-1501504905252-473c47e087f8?w=800&q=80",
    "https://images.unsplash.com/photo-1498050108023-c5249f4df085?w=800&q=80",
    "https://images.unsplash.com/photo-1504639725590-34d0984388bd?w=800&q=80",
    "https://images.unsplash.com/photo-1517694712202-14dd9538aa97?w=800&q=80",
    "https://images.unsplash.com/photo-1522202176988-66273c2fd55f?w=800&q=80",
]

SUBJECTS = [
    ("Data Science", "Python analytics, statistics, dashboards, and data storytelling", "bar-chart"),
    ("AI & Machine Learning", "Machine learning, neural networks, and applied AI", "brain"),
    ("Fullstack Development", "Frontend, backend, APIs, and modern web apps", "code"),
    ("Cloud Computing", "Cloud architecture, containers, DevOps, and deployment", "cloud"),
    ("Cybersecurity", "Security fundamentals, defensive systems, and ethical hacking", "shield"),
    ("Mobile Development", "React Native, Flutter, and mobile product delivery", "smartphone"),
    ("UI/UX Design", "Research, wireframes, visual systems, and product design", "layout"),
    ("Data Engineering", "Pipelines, warehouses, orchestration, and big data systems", "database"),
]

COURSE_TOPICS = {
    "Data Science": [
        "Python Data Analysis", "Statistics for Analysts", "SQL Analytics", "Business Intelligence",
        "Data Visualization", "Predictive Analytics", "Excel to Python", "Analytics Capstone",
        "Dashboard Design", "Experiment Analysis", "Pandas Deep Dive", "Applied Regression",
    ],
    "AI & Machine Learning": [
        "Machine Learning Essentials", "Deep Learning Foundations", "NLP Fundamentals",
        "Computer Vision", "Recommendation Systems", "Model Evaluation", "AI Product Thinking",
        "MLOps Basics", "Neural Networks", "Prompt Engineering", "Applied Classification",
        "Feature Engineering",
    ],
    "Fullstack Development": [
        "React Fundamentals", "Next.js Applications", "Node.js APIs", "TypeScript Mastery",
        "Database-backed Web Apps", "Authentication Systems", "GraphQL APIs", "Testing Web Apps",
        "Frontend Architecture", "Backend Services", "Web Performance", "Fullstack Capstone",
    ],
    "Cloud Computing": [
        "Cloud Architecture", "Docker Essentials", "Kubernetes Basics", "AWS Deployment",
        "Serverless Systems", "CI/CD Pipelines", "Infrastructure as Code", "Cloud Monitoring",
        "Distributed Systems", "DevOps Practices", "Linux for Cloud", "Scalable APIs",
    ],
    "Cybersecurity": [
        "Security Fundamentals", "Web App Security", "Network Defense", "Threat Modeling",
        "Incident Response", "Identity Security", "Secure Coding", "Cloud Security",
        "Penetration Testing", "Security Monitoring", "Cryptography Basics", "API Security",
    ],
    "Mobile Development": [
        "React Native Basics", "Flutter Foundations", "Mobile UI Patterns", "Offline-first Apps",
        "Mobile Navigation", "Expo Applications", "State Management", "Mobile Testing",
        "App Store Release", "Native Modules", "Mobile Performance", "Cross-platform Capstone",
    ],
    "UI/UX Design": [
        "UX Research", "Wireframing", "Figma Components", "Design Systems", "Interaction Design",
        "Accessibility Design", "Product Prototyping", "Usability Testing", "Visual Hierarchy",
        "Mobile UX", "Portfolio Case Studies", "Design Critique",
    ],
    "Data Engineering": [
        "Data Engineering Essentials", "ETL Pipelines", "Airflow Orchestration", "Data Warehousing",
        "Spark Fundamentals", "Streaming Data", "dbt Analytics Engineering", "Data Modeling",
        "Lakehouse Architecture", "Pipeline Monitoring", "Big Data Systems", "Data Quality",
    ],
}

PERSONAS = [
    ("rec-data-01@example.com", "Nour Data", ["Data Science", "AI & Machine Learning", "Data Engineering"]),
    ("rec-data-02@example.com", "Sara Analyst", ["Data Science", "Data Engineering", "AI & Machine Learning"]),
    ("rec-ai-01@example.com", "Karim AI", ["AI & Machine Learning", "Data Science", "Cloud Computing"]),
    ("rec-ai-02@example.com", "Hana ML", ["AI & Machine Learning", "Data Engineering", "Data Science"]),
    ("rec-web-01@example.com", "Omar Web", ["Fullstack Development", "UI/UX Design", "Cloud Computing"]),
    ("rec-web-02@example.com", "Maya Fullstack", ["Fullstack Development", "Mobile Development", "Cloud Computing"]),
    ("rec-cloud-01@example.com", "Adel Cloud", ["Cloud Computing", "Cybersecurity", "Fullstack Development"]),
    ("rec-security-01@example.com", "Rana Security", ["Cybersecurity", "Cloud Computing", "Fullstack Development"]),
    ("rec-mobile-01@example.com", "Liam Mobile", ["Mobile Development", "UI/UX Design", "Fullstack Development"]),
    ("rec-design-01@example.com", "Mona Design", ["UI/UX Design", "Mobile Development", "Fullstack Development"]),
]


def password_hash(password: str = "password123") -> str:
    if password != "password123":
        raise ValueError("This seeder only provides the precomputed password123 hash.")
    return PASSWORD123_BCRYPT


def table_columns(cur, schema: str, table: str) -> set[str]:
    cur.execute(
        """
        SELECT column_name
        FROM information_schema.columns
        WHERE table_schema = %s AND table_name = %s
        """,
        (schema, table),
    )
    return {row[0] for row in cur.fetchall()}


def insert_dynamic(cur, table: str, rows: Sequence[Dict], allowed_columns: Iterable[str]) -> None:
    if not rows:
        return
    allowed = set(allowed_columns)
    columns = [key for key in rows[0].keys() if key in allowed]
    values = [[row.get(col) for col in columns] for row in rows]
    quoted = ", ".join(f'"{col}"' if any(c.isupper() for c in col) else col for col in columns)
    execute_values(cur, f"INSERT INTO {table} ({quoted}) VALUES %s", values)


def reset_database(cur) -> None:
    print("Resetting auth/course/recommendation data...")
    cur.execute(
        """
        SELECT table_schema || '.' || table_name
        FROM information_schema.tables
        WHERE table_schema IN ('auth', 'public')
        """
    )
    existing = {row[0] for row in cur.fetchall()}
    tables = [
        'auth."Session"', 'auth."UserDevice"', 'auth."AuthProvider"',
        'auth."UserPreference"', 'auth."ParentChildLink"', 'auth."ParentLinkRequest"',
        'auth."UnlinkRequest"', 'auth."LocationHistory"', 'auth."CourseEnrollment"',
        'auth."UserInterest"', 'auth."Interest"', 'auth."User"',
        "public.recommendation_history", "public.user_clusters", "public.cluster_metadata",
        "public.preview_watch_events", "public.user_preview_progress",
        "public.lesson_watch_events", "public.user_lesson_progress", "public.user_course_analytics",
        "public.enrollment_periods", "public.enrollments", "public.lessons",
        "public.course_assistants", "public.course_ratings", "public.teacher_ratings",
        "public.courses", "public.subjects",
    ]
    for table in tables:
        lookup = table.replace('"', "")
        if lookup not in existing:
            print(f"  skipped {table}")
            continue
        cur.execute(f"TRUNCATE {table} CASCADE")


def create_subjects_and_interests(cur, now: datetime) -> Dict[str, str]:
    subject_rows = []
    interest_rows = []
    subject_ids = {}
    for name, description, icon in SUBJECTS:
        sid = str(uuid.uuid4())
        subject_ids[name] = sid
        subject_rows.append((sid, name, description, icon, now, now))
        interest_rows.append((sid, name, now, now))

    execute_values(
        cur,
        "INSERT INTO public.subjects (id, name, description, icon, created_at, updated_at) VALUES %s",
        subject_rows,
    )
    execute_values(
        cur,
        'INSERT INTO auth."Interest" (id, name, "createdAt", "updatedAt") VALUES %s',
        interest_rows,
    )
    return subject_ids


def create_users(cur, now: datetime, subject_ids: Dict[str, str]) -> Tuple[str, Dict[str, List[str]]]:
    print("Creating teacher + 10 recommendation test students...")
    pwd = password_hash()
    teacher_id = str(uuid.uuid4())
    cur.execute(
        'INSERT INTO auth."User" '
        '(id, name, username, email, password, role, "onboardingCompleted", verified, "createdAt", "updatedAt") '
        "VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)",
        (teacher_id, "Recommendation Teacher", "rec_teacher", "rec-teacher@example.com", pwd, "TEACHER", True, True, now, now),
    )

    user_interests: Dict[str, List[str]] = {}
    user_rows = []
    interest_rows = []
    for idx, (email, name, interests) in enumerate(PERSONAS, start=1):
        uid = str(uuid.uuid4())
        username = f"rec_student_{idx:02d}"
        user_rows.append((uid, name, username, email, pwd, "STUDENT", True, True, now, now))
        user_interests[uid] = interests
        for interest in interests:
            interest_rows.append((uid, subject_ids[interest], now))

    execute_values(
        cur,
        'INSERT INTO auth."User" '
        '(id, name, username, email, password, role, "onboardingCompleted", verified, "createdAt", "updatedAt") '
        "VALUES %s",
        user_rows,
    )
    execute_values(
        cur,
        'INSERT INTO auth."UserInterest" ("userId", "interestId", "createdAt") VALUES %s',
        interest_rows,
    )
    return teacher_id, user_interests


def create_courses(cur, now: datetime, teacher_id: str, subject_ids: Dict[str, str]) -> Dict[str, Dict]:
    print("Creating 100 coherent courses...")
    course_columns = table_columns(cur, "public", "courses")
    course_rows = []
    course_meta = {}
    prefixes = ["Essential", "Complete", "Modern", "Professional", "Advanced", "Practical", "Deep Dive"]
    suffixes = ["Bootcamp", "Masterclass", "Workshop", "Essentials", "Project Lab", "Handbook"]

    subjects_cycle = [name for name, _, _ in SUBJECTS]
    for idx in range(100):
        subject = subjects_cycle[idx % len(subjects_cycle)]
        topics = COURSE_TOPICS[subject]
        topic = topics[(idx // len(subjects_cycle)) % len(topics)]
        title = f"{random.choice(prefixes)} {topic} {random.choice(suffixes)}"
        course_id = str(uuid.uuid4())
        total_lessons = random.randint(6, 14)
        is_paid = idx % 4 != 0
        delivery = "ONLINE" if idx % 5 != 1 else "OFFLINE"
        row = {
            "id": course_id,
            "title": title,
            "description": f"{title} teaches {topic.lower()} as part of the {subject} learning path.",
            "subject_id": subject_ids[subject],
            "teacher_id": teacher_id,
            "delivery_type": delivery,
            "location_name": "Cairo Learning Hub" if delivery == "OFFLINE" else None,
            "location_lat": 30.0444 if delivery == "OFFLINE" else None,
            "location_lng": 31.2357 if delivery == "OFFLINE" else None,
            "geofence_radius_m": 200 if delivery == "OFFLINE" else 100,
            "total_lessons": total_lessons,
            "attendance_window_minutes": 30,
            "price": 350.0 if is_paid else 0.0,
            "currency": "EGP",
            "is_paid": is_paid,
            "billing_type": "ONE_TIME" if idx % 3 else "MONTHLY",
            "free_trial_lessons": 2 if is_paid else 0,
            "status": "ACTIVE",
            "attendance_weight": 0.3,
            "reminder_intervals": "30,15,5",
            "course_image": THUMBS[idx % len(THUMBS)],
            "group_image": THUMBS[(idx + 1) % len(THUMBS)],
            "preview_video_url": "https://res.cloudinary.com/demo/video/upload/sp_auto/sea_turtle.m3u8",
            "preview_video_public_id": "demo/sea_turtle",
            "created_at": now,
            "updated_at": now,
        }
        course_rows.append(row)
        course_meta[course_id] = {"subject": subject, "total_lessons": total_lessons, "title": title}

    insert_dynamic(cur, "public.courses", course_rows, course_columns)
    return course_meta


def create_lessons(cur, now: datetime, course_meta: Dict[str, Dict]) -> Dict[str, List[str]]:
    print("Creating lessons...")
    lesson_columns = table_columns(cur, "public", "lessons")
    lesson_rows = []
    lessons_by_course = {}
    for course_index, (course_id, meta) in enumerate(course_meta.items()):
        lessons_by_course[course_id] = []
        for lesson_no in range(1, meta["total_lessons"] + 1):
            lesson_id = str(uuid.uuid4())
            lessons_by_course[course_id].append(lesson_id)
            is_online = course_index % 5 != 1
            duration_minutes = random.choice([20, 25, 30, 35, 40])
            row = {
                "id": lesson_id,
                "course_id": course_id,
                "title": f"Lesson {lesson_no}: {meta['title']} Part {lesson_no}",
                "description": f"Guided lesson {lesson_no} for {meta['subject']}.",
                "lesson_number": lesson_no,
                "scheduled_at": now + timedelta(days=lesson_no),
                "starts_at": None,
                "ends_at": None,
                "duration_minutes": duration_minutes,
                "duration": duration_minutes * 60,
                "status": "SCHEDULED",
                "delivery_type": "ONLINE" if is_online else "OFFLINE",
                "is_free": lesson_no <= 2,
                "video_url": "https://res.cloudinary.com/demo/video/upload/dog.mp4" if is_online else None,
                "video_public_id": "demo/dog" if is_online else None,
                "materials_url": "https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf" if is_online else None,
                "thumbnail_url": THUMBS[lesson_no % len(THUMBS)],
                "reminders_sent": "",
                "created_at": now,
                "updated_at": now,
            }
            lesson_rows.append(row)

    insert_dynamic(cur, "public.lessons", lesson_rows, lesson_columns)
    return lessons_by_course


def engagement_score(completion_pct: float, avg_watch_pct: float, last_activity: datetime, now: datetime) -> float:
    recency = max(0.0, 100.0 - ((now - last_activity).days / 30.0) * 100.0)
    score = 0.45 * completion_pct + 0.35 * avg_watch_pct + 0.20 * recency
    return round(min(100.0, max(0.0, score)), 2)


def create_learning_activity(cur, now: datetime, user_interests: Dict[str, List[str]], course_meta: Dict[str, Dict], lessons_by_course: Dict[str, List[str]]) -> None:
    print("Creating enrollments + realistic watch analytics...")
    courses_by_subject: Dict[str, List[str]] = {}
    for cid, meta in course_meta.items():
        courses_by_subject.setdefault(meta["subject"], []).append(cid)

    enrollment_rows = []
    analytics_rows = []
    progress_rows = []
    event_rows = []

    for user_index, (user_id, interests) in enumerate(user_interests.items()):
        primary, secondary, tertiary = interests
        selected = []
        selected += courses_by_subject[primary][:6]
        selected += courses_by_subject[secondary][2:6]
        selected += courses_by_subject[tertiary][4:6]
        # Add a little noise so clusters are not perfectly identical.
        noise_subject = SUBJECTS[(user_index + 3) % len(SUBJECTS)][0]
        selected += courses_by_subject[noise_subject][:1]
        selected = list(dict.fromkeys(selected))[:12]

        for course_offset, course_id in enumerate(selected):
            meta = course_meta[course_id]
            total_lessons = meta["total_lessons"]
            strong = meta["subject"] == primary
            medium = meta["subject"] == secondary
            completion_target = random.uniform(0.78, 1.0) if strong else random.uniform(0.45, 0.76) if medium else random.uniform(0.18, 0.42)
            lessons_started = max(1, min(total_lessons, int(round(total_lessons * min(1.0, completion_target + 0.12)))))
            lessons_completed = max(0, min(lessons_started, int(round(total_lessons * completion_target))))
            completion_pct = round((lessons_completed / total_lessons) * 100.0, 2)
            last_activity = now - timedelta(days=random.randint(0, 21), hours=random.randint(0, 23))

            enrollment_rows.append((str(uuid.uuid4()), course_id, user_id, True, True, now - timedelta(days=30 + course_offset), now))

            total_watch = 0
            lesson_pcts = []
            for lesson_idx, lesson_id in enumerate(lessons_by_course[course_id][:lessons_started], start=1):
                duration = 30 * 60
                if lesson_idx <= lessons_completed:
                    pct = random.uniform(90, 100)
                    completed = True
                else:
                    pct = random.uniform(25, 75)
                    completed = False
                watched = int(duration * pct / 100.0)
                watch_count = random.randint(1, 3 if strong else 2)
                total_watch += watched
                lesson_pcts.append(pct)
                watched_at = last_activity - timedelta(days=max(0, lessons_started - lesson_idx))
                progress_rows.append((
                    str(uuid.uuid4()), lesson_id, user_id, watched, watched, watch_count,
                    round(pct, 2), completed, watched_at, watched_at, now, now,
                ))
                event_rows.append((
                    str(uuid.uuid4()), lesson_id, user_id, watched, watched, completed,
                    random.choice(["MOBILE", "DESKTOP", "TABLET"]), watched_at,
                ))

            avg_watch_pct = round(sum(lesson_pcts) / len(lesson_pcts), 2) if lesson_pcts else 0
            analytics_rows.append((
                str(uuid.uuid4()), course_id, user_id, total_watch, lessons_started,
                lessons_completed, total_lessons, completion_pct, avg_watch_pct,
                engagement_score(completion_pct, avg_watch_pct, last_activity, now),
                last_activity, now, now,
            ))

    execute_values(
        cur,
        "INSERT INTO public.enrollments (id, course_id, user_id, is_active, is_paid, enrolled_at, updated_at) VALUES %s",
        enrollment_rows,
    )
    execute_values(
        cur,
        "INSERT INTO public.user_lesson_progress "
        "(id, lesson_id, user_id, total_watch_time, max_position, watch_count, completion_pct, is_completed, "
        " first_watched_at, last_watched_at, created_at, updated_at) VALUES %s",
        progress_rows,
    )
    execute_values(
        cur,
        "INSERT INTO public.lesson_watch_events "
        "(id, lesson_id, user_id, watched_seconds, last_position, completed, device_type, created_at) VALUES %s",
        event_rows,
    )
    execute_values(
        cur,
        "INSERT INTO public.user_course_analytics "
        "(id, course_id, user_id, total_watch_time, lessons_started, lessons_completed, total_lessons, "
        " completion_pct, avg_lesson_watch_pct, engagement_score, last_activity_at, created_at, updated_at) VALUES %s",
        analytics_rows,
    )


def clear_ai_state(cur) -> None:
    cur.execute(
        """
        SELECT table_schema || '.' || table_name
        FROM information_schema.tables
        WHERE table_schema = 'public'
        """
    )
    existing = {row[0] for row in cur.fetchall()}
    for table in ["public.recommendation_history", "public.user_clusters", "public.cluster_metadata"]:
        if table in existing:
            cur.execute(f"TRUNCATE {table} CASCADE")


def main() -> None:
    conn = psycopg2.connect(DB_URL)
    conn.autocommit = False
    cur = conn.cursor()
    now = datetime.now(timezone.utc).replace(microsecond=0)
    try:
        reset_database(cur)
        subject_ids = create_subjects_and_interests(cur, now)
        teacher_id, user_interests = create_users(cur, now, subject_ids)
        course_meta = create_courses(cur, now, teacher_id, subject_ids)
        lessons_by_course = create_lessons(cur, now, course_meta)
        create_learning_activity(cur, now, user_interests, course_meta, lessons_by_course)
        clear_ai_state(cur)
        conn.commit()

        print("\nSeed complete.")
        print("Teacher: rec-teacher@example.com / password123")
        print("Students:")
        for email, name, interests in PERSONAS:
            print(f"  {email:28s} / password123 / {name} / {', '.join(interests)}")
        print("\nNext:")
        print("  docker compose restart recommendation-service")
        print("  POST {{gateway_url}}/api/v1/recommendations/clusters/rebuild")
        print("  POST {{gateway_url}}/api/v1/recommendations/refresh")
    except Exception as exc:
        conn.rollback()
        print(f"Seed failed: {exc}")
        raise
    finally:
        cur.close()
        conn.close()


if __name__ == "__main__":
    main()
