import uuid
import random
from datetime import datetime, timedelta
import psycopg2
from psycopg2.extras import execute_values
import bcrypt
import os

# Configuration
DB_URL = "postgresql://graduation:graduation_secret@localhost:5432/graduation"

def get_password_hash(password: str):
    salt = bcrypt.gensalt()
    hashed = bcrypt.hashpw(password.encode('utf-8'), salt)
    return hashed.decode('utf-8')

def seed():
    print("Starting Seeding with VALID Cloudinary URLs...")
    conn = psycopg2.connect(DB_URL)
    cur = conn.cursor()

    try:
        now = datetime.now()
        
        print("Cleaning existing data...")
        cur.execute("""
            TRUNCATE auth."Interest", auth."UserInterest", auth."ParentChildLink",
            public.courses, public.subjects, public.enrollments, 
            public.user_course_analytics, public.user_lesson_progress, 
            public.lessons, public.enrollment_periods CASCADE;
        """)

        # 1. Subjects
        subjects = [
            (str(uuid.uuid4()), "Mobile Development", "React Native & Flutter", "smartphone"),
            (str(uuid.uuid4()), "Cloud Architecture", "AWS & Docker", "cloud"),
            (str(uuid.uuid4()), "UI/UX Design", "Figma & Design Systems", "layers"),
            (str(uuid.uuid4()), "Data Science", "Python & AI", "database")
        ]
        execute_values(cur, "INSERT INTO public.subjects (id, name, description, icon) VALUES %s", subjects)

        # 2. Get or Create Teacher & Student with stable IDs
        def get_or_create_user(email, name, username, role):
            cur.execute("SELECT id FROM auth.\"User\" WHERE email = %s", (email,))
            row = cur.fetchone()
            if row:
                return row[0]
            
            uid = str(uuid.uuid4())
            cur.execute(
                """INSERT INTO auth.\"User\" (id, name, username, email, password, role, \"onboardingCompleted\", verified, \"createdAt\", \"updatedAt\") 
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)""",
                (uid, name, username, email, get_password_hash("password123"), role, True, True, now, now)
            )
            return uid

        teacher_id = get_or_create_user("teacher@example.com", "Prof. Omar", "omar_teacher", "TEACHER")
        student_id = get_or_create_user("student@example.com", "Test Student", "student_user", "STUDENT")
        parent_id = get_or_create_user("parent@example.com", "Guardian User", "parent_user", "PARENT")

        print(f"Teacher ID: {teacher_id}")
        print(f"Student ID: {student_id}")
        print(f"Parent ID: {parent_id}")

        # 2.2. Link parent to student
        print("Linking parent to student...")
        cur.execute(
            "INSERT INTO auth.\"ParentChildLink\" (id, \"parentId\", \"childId\", \"createdAt\") VALUES (%s, %s, %s, %s) ON CONFLICT DO NOTHING",
            (str(uuid.uuid4()), parent_id, student_id, now)
        )

        # 2.5. Interests
        print("Seeding Interests & Assigning to student...")
        interests = [(s[0], s[1], now, now) for s in subjects]
        execute_values(cur, "INSERT INTO auth.\"Interest\" (id, name, \"createdAt\", \"updatedAt\") VALUES %s", interests)

        selected_interests = random.sample(subjects, 3)
        student_interests = [(student_id, si[0], now) for si in selected_interests]
        execute_values(cur, "INSERT INTO auth.\"UserInterest\" (\"userId\", \"interestId\", \"createdAt\") VALUES %s", student_interests)

        # 3. Valid Assets Pool
        VALID_THUMBS = [
            "https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=800&q=80",
            "https://images.unsplash.com/photo-1501504905252-473c47e087f8?w=800&q=80",
            "https://images.unsplash.com/photo-1498050108023-c5249f4df085?w=800&q=80",
            "https://images.unsplash.com/photo-1504639725590-34d0984388bd?w=800&q=80",
            "https://images.unsplash.com/photo-1517694712202-14dd9538aa97?w=800&q=80",
            "https://images.unsplash.com/photo-1522202176988-66273c2fd55f?w=800&q=80"
        ]
        VALID_HLS_VIDEOS = [
            ("https://res.cloudinary.com/demo/video/upload/sp_auto/elephants.m3u8", "demo/elephants"),
            ("https://res.cloudinary.com/demo/video/upload/sp_auto/sea_turtle.m3u8", "demo/sea_turtle"),
            ("https://res.cloudinary.com/demo/video/upload/sp_auto/rock_climber.m3u8", "demo/rock_climber")
        ]
        VALID_MP4 = "https://res.cloudinary.com/demo/video/upload/dog.mp4"

        # 4. Courses
        course_prefixes = ["Mastering", "Intro to", "Advanced", "The Art of", "Modern", "Professional", "Complete", "Deep Dive:", "Essential"]
        course_topics = ["React Native", "Docker", "Python", "Kubernetes", "UI/UX", "Machine Learning", "Cloud Architecture", "Go Programming", "Cybersecurity", "Blockchain", "Data Engineering", "Next.js"]
        course_suffixes = ["for Experts", "Masterclass", "Bootcamp", "Essentials", "Simplified", "Course 2024", "Handbook", "Workshop"]

        courses_to_create = []
        for i in range(20):
            prefix = random.choice(course_prefixes)
            topic = random.choice(course_topics)
            suffix = random.choice(course_suffixes)
            title = f"{prefix} {topic} {suffix}"
            
            # Coverage for all cases
            delivery = "ONLINE" if i % 2 == 0 else "OFFLINE"
            billing = "ONE_TIME" if i % 3 == 0 else "MONTHLY"
            is_paid = (i % 4 != 0) # Every 4th course is free
            price = round(random.uniform(50.0, 1000.0), 2) if is_paid else 0.0
            
            s_idx = random.randint(0, len(subjects) - 1)
            enroll = (i < 5) # Enroll in the first 5 for testing
            total_less = random.randint(3, 15)
            
            thumb = random.choice(VALID_THUMBS)
            v_url, v_id = random.choice(VALID_HLS_VIDEOS)
            
            courses_to_create.append({
                "id": str(uuid.uuid4()),
                "title": title,
                "delivery": delivery,
                "billing": billing,
                "price": price,
                "is_paid": is_paid,
                "subject_idx": s_idx,
                "should_enroll": enroll,
                "total_lessons": total_less,
                "thumb": thumb,
                "v_url": v_url,
                "v_id": v_id
            })
        course_data = []
        for c in courses_to_create:
            course_data.append((
                c["id"], c["title"], f"Learn {c['title']} with real world projects.", subjects[c["subject_idx"]][0], teacher_id, c["delivery"],
                "Cairo Digital District" if c["delivery"] == "OFFLINE" else None,
                30.0444 if c["delivery"] == "OFFLINE" else None, 31.2357 if c["delivery"] == "OFFLINE" else None,
                150, c["total_lessons"], 15, c["price"], c["is_paid"], "ACTIVE", 0.3, "EGP", c["billing"],
                random.choice(["15,10,5", "30,15", "60,30,5", "15"]), # NEW: reminder_intervals
                c["thumb"], c["v_url"], c["v_id"], now, now
            ))

        execute_values(cur, """
            INSERT INTO public.courses (
                id, title, description, subject_id, teacher_id, delivery_type, 
                location_name, location_lat, location_lng, geofence_radius_m, 
                total_lessons, attendance_window_minutes, price, is_paid, 
                status, attendance_weight, currency, billing_type, 
                reminder_intervals, course_image, preview_video_url, preview_video_public_id,
                created_at, updated_at
            ) VALUES %s
        """, course_data)

        # 5. Lessons
        lesson_data = []
        for c in courses_to_create:
            for n in range(1, c["total_lessons"] + 1):
                l_id = str(uuid.uuid4())
                is_free = (n <= 2) # First 2 lessons are always free trial
                is_online = (c["delivery"] == "ONLINE")
                
                # Use dog.mp4 for lessons
                vid = VALID_MP4 if is_online else None
                mat = "https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf" if is_online else None
                
                lesson_data.append((
                    l_id, c["id"], f"Part {n}: Moving forward", f"Module {n} description for {c['title']}",
                    n, now + timedelta(days=n), 60, "SCHEDULED", c["delivery"], is_free,
                    vid, "demo/dog" if vid else None, mat, "", now, now # NEW: "" for reminders_sent
                ))

        execute_values(cur, """
            INSERT INTO public.lessons (
                id, course_id, title, description, lesson_number, 
                scheduled_at, duration_minutes, status, delivery_type, 
                is_free, video_url, video_public_id, materials_url,
                reminders_sent, created_at, updated_at
            ) VALUES %s
        """, lesson_data)

        # 6. Enrollments
        for c in courses_to_create:
            if c["should_enroll"]:
                e_id = str(uuid.uuid4())
                cur.execute(
                    "INSERT INTO public.enrollments (id, course_id, user_id, is_active, is_paid, enrolled_at) VALUES (%s, %s, %s, %s, %s, %s)",
                    (e_id, c["id"], student_id, True, True, now)
                )
                if c["billing"] == "MONTHLY":
                    cur.execute(
                        "INSERT INTO public.enrollment_periods (id, enrollment_id, period_key, is_paid, paid_at) VALUES (%s, %s, %s, %s, %s)",
                        (str(uuid.uuid4()), e_id, now.strftime("%Y-%m"), True, now)
                    )

        # 7. SPECIAL TEST SCENARIO (As per USER_REQUEST)
        print("Creating 5 Special Test Scenario Courses...")
        for i in range(1, 6):
            test_course_id = str(uuid.uuid4())
            test_lesson_id = str(uuid.uuid4())
            # Schedule staggered lessons: 25, 35, 45, 55, 65 mins from now
            offset_mins = 25 + ((i-1) * 10)
            lesson_time = now + timedelta(minutes=offset_mins)
            
            # Course with 10km radius (relaxed) and 15,10,5 reminders
            cur.execute("""
                INSERT INTO public.courses (
                    id, title, description, subject_id, teacher_id, delivery_type, 
                    location_name, location_lat, location_lng, geofence_radius_m, 
                    total_lessons, attendance_window_minutes, price, is_paid, 
                    status, attendance_weight, currency, billing_type, 
                    reminder_intervals, course_image, preview_video_url, preview_video_public_id,
                    created_at, updated_at
                ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            """, (
                test_course_id, f"OFFLINE LIVE TEST #{i} - Reminders & Attendance", f"Test course #{i} for offline reminders and attendance.",
                subjects[0][0], teacher_id, "OFFLINE", f"Test Location {i}", 30.0444, 31.2357, 10000, 
                1, 30, 99.99, True, "ACTIVE", 0.5, "EGP", "ONE_TIME", "15,10,5", 
                VALID_THUMBS[i % len(VALID_THUMBS)], VALID_HLS_VIDEOS[0][0], VALID_HLS_VIDEOS[0][1], now, now
            ))

            # Lesson for this course
            cur.execute("""
                INSERT INTO public.lessons (
                    id, course_id, title, description, lesson_number, 
                    scheduled_at, duration_minutes, status, delivery_type, 
                    is_free, reminders_sent, created_at, updated_at
                ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            """, (
                test_lesson_id, test_course_id, f"Live Demo Lesson #{i}", f"Join this demo #{i} to test notifications.",
                1, lesson_time, 60, "SCHEDULED", "OFFLINE", False, "", now, now
            ))

            # Enroll student in this test course
            test_enroll_id = str(uuid.uuid4())
            cur.execute(
                "INSERT INTO public.enrollments (id, course_id, user_id, is_active, is_paid, enrolled_at) VALUES (%s, %s, %s, %s, %s, %s)",
                (test_enroll_id, test_course_id, student_id, True, True, now)
            )

        conn.commit()
        print("SUCCESS: Data seeded with valid URLs.")
        print("Teacher: teacher@example.com / password123")
        print("Student: student@example.com / password123")
        print("Parent: parent@example.com / password123")

    except Exception as e:
        print(f"ERROR: {e}")
        conn.rollback()
    finally:
        cur.close()
        conn.close()

if __name__ == "__main__":
    seed()
