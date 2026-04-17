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
            TRUNCATE auth."User", auth."Interest", auth."UserInterest", 
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

        # 2. Users
        teacher_id = str(uuid.uuid4())
        student_id = str(uuid.uuid4())
        
        cur.execute(
            """INSERT INTO auth.\"User\" (id, name, username, email, password, role, \"onboardingCompleted\", verified, \"createdAt\", \"updatedAt\") 
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)""",
            (teacher_id, "Prof. Omar", "omar_teacher", "teacher@example.com", get_password_hash("password123"), "TEACHER", True, True, now, now)
        )
        cur.execute(
            """INSERT INTO auth.\"User\" (id, name, username, email, password, role, \"onboardingCompleted\", verified, \"createdAt\", \"updatedAt\") 
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)""",
            (student_id, "Test Student", "student_user", "student@example.com", get_password_hash("password123"), "STUDENT", True, True, now, now)
        )

        # 3. Valid Assets
        VALID_THUMB = "https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=800&q=80"
        VALID_HLS = "https://res.cloudinary.com/demo/video/upload/sp_auto/elephants.m3u8"
        VALID_MP4 = "https://res.cloudinary.com/demo/video/upload/dog.mp4"

        # 4. Courses
        courses_to_create = [
            (str(uuid.uuid4()), "React Native Mastery", "ONLINE", "MONTHLY", 250.0, 0, True),
            (str(uuid.uuid4()), "Docker in Production", "OFFLINE", "ONE_TIME", 600.0, 1, True), 
            (str(uuid.uuid4()), "Figma Advanced", "ONLINE", "ONE_TIME", 400.0, 2, False),
            (str(uuid.uuid4()), "Python for Data Science", "ONLINE", "MONTHLY", 180.0, 3, False)
        ]

        course_data = []
        for c_id, title, delivery, billing, price, s_idx, enroll in courses_to_create:
            course_data.append((
                c_id, title, f"Learn {title} with real world projects.", subjects[s_idx][0], teacher_id, delivery,
                "Cairo Digital District" if delivery == "OFFLINE" else None,
                30.0444 if delivery == "OFFLINE" else None, 31.2357 if delivery == "OFFLINE" else None,
                150, 5, 15, price, True, "ACTIVE", 0.3, "EGP", billing,
                VALID_THUMB, VALID_HLS, "demo/elephants", now, now
            ))

        execute_values(cur, """
            INSERT INTO public.courses (
                id, title, description, subject_id, teacher_id, delivery_type, 
                location_name, location_lat, location_lng, geofence_radius_m, 
                total_lessons, attendance_window_minutes, price, is_paid, 
                status, attendance_weight, currency, billing_type, 
                course_image, preview_video_url, preview_video_public_id,
                created_at, updated_at
            ) VALUES %s
        """, course_data)

        # 5. Lessons
        lesson_data = []
        for c_id, _, delivery, _, _, _, _ in courses_to_create:
            for n in range(1, 6):
                l_id = str(uuid.uuid4())
                is_free = (n == 1)
                is_online = (delivery == "ONLINE")
                
                # Use dog.mp4 for lessons
                vid = VALID_MP4 if is_online else None
                mat = "https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf" if is_online else None
                
                lesson_data.append((
                    l_id, c_id, f"Part {n}: Moving forward", f"Module {n} description",
                    n, now + timedelta(days=n), 60, "SCHEDULED", delivery, is_free,
                    vid, "demo/dog" if vid else None, mat, now, now
                ))

        execute_values(cur, """
            INSERT INTO public.lessons (
                id, course_id, title, description, lesson_number, 
                scheduled_at, duration_minutes, status, delivery_type, 
                is_free, video_url, video_public_id, materials_url,
                created_at, updated_at
            ) VALUES %s
        """, lesson_data)

        # 6. Enrollments
        for c_id, _, _, billing, _, _, should_enroll in courses_to_create:
            if should_enroll:
                e_id = str(uuid.uuid4())
                cur.execute(
                    "INSERT INTO public.enrollments (id, course_id, user_id, is_active, is_paid, enrolled_at) VALUES (%s, %s, %s, %s, %s, %s)",
                    (e_id, c_id, student_id, True, True, now)
                )
                if billing == "MONTHLY":
                    cur.execute(
                        "INSERT INTO public.enrollment_periods (id, enrollment_id, period_key, is_paid, paid_at) VALUES (%s, %s, %s, %s, %s)",
                        (str(uuid.uuid4()), e_id, now.strftime("%Y-%m"), True, now)
                    )

        conn.commit()
        print("SUCCESS: Data seeded with valid URLs.")
        print("User: student@example.com / password123")

    except Exception as e:
        print(f"ERROR: {e}")
        conn.rollback()
    finally:
        cur.close()
        conn.close()

if __name__ == "__main__":
    seed()
