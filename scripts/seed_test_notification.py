import uuid
from datetime import datetime, timedelta, timezone
import psycopg2
import os

# Configuration
DB_URL = "postgresql://graduation:graduation_secret@localhost:5432/graduation"

def seed_test_lesson():
    print("Seeding test offline course and lesson (UTC corrected)...")
    conn = psycopg2.connect(DB_URL)
    cur = conn.cursor()

    try:
        # Use UTC to match the Go backend's clock
        now_utc = datetime.now(timezone.utc)
        
        # 1. Get Teacher & Student
        cur.execute("SELECT id FROM auth.\"User\" WHERE email = %s", ("teacher@example.com",))
        teacher_id = cur.fetchone()
        if not teacher_id:
            print("Error: Teacher not found. Please run seed_data.py first.")
            return
        teacher_id = teacher_id[0]

        cur.execute("SELECT id FROM auth.\"User\" WHERE email = %s", ("student@example.com",))
        student_id = cur.fetchone()
        if not student_id:
            print("Error: Student not found. Please run seed_data.py first.")
            return
        student_id = student_id[0]

        # 2. Get a subject
        cur.execute("SELECT id FROM public.subjects LIMIT 1")
        subject_id = cur.fetchone()[0]

        # 3. Create Course
        course_id = str(uuid.uuid4())
        lesson_time_utc = now_utc + timedelta(minutes=30)
        
        print(f"Current UTC: {now_utc.strftime('%H:%M:%S')}")
        print(f"Scheduled UTC: {lesson_time_utc.strftime('%H:%M:%S')}")
        print(f"This should appear as 30 mins away in the app (Egypt time: {(lesson_time_utc + timedelta(hours=2)).strftime('%H:%M:%S')})")

        cur.execute("""
            INSERT INTO public.courses (
                id, title, description, subject_id, teacher_id, delivery_type, 
                location_name, location_lat, location_lng, geofence_radius_m, 
                total_lessons, attendance_window_minutes, price, is_paid, 
                status, attendance_weight, currency, billing_type, 
                reminder_intervals, course_image, created_at, updated_at
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        """, (
            course_id, "UTC TEST - OFFLINE", "Course for testing 30min notifications (UTC sync)",
            subject_id, teacher_id, "OFFLINE", "VeriQ Test Center", 30.0444, 31.2357, 20000000, 
            1, 30, 0.0, False, "ACTIVE", 0.5, "EGP", "ONE_TIME", "30,15,5", 
            "https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=800&q=80", now_utc, now_utc
        ))

        # 4. Create Lesson
        lesson_id = str(uuid.uuid4())
        cur.execute("""
            INSERT INTO public.lessons (
                id, course_id, title, description, lesson_number, 
                scheduled_at, duration_minutes, status, delivery_type, 
                is_free, reminders_sent, created_at, updated_at
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        """, (
            lesson_id, course_id, "UTC Sync Lesson", "Notification test lesson with UTC sync",
            1, lesson_time_utc, 60, "SCHEDULED", "OFFLINE", True, "", now_utc, now_utc
        ))

        # 5. Enroll Student
        cur.execute(
            "INSERT INTO public.enrollments (id, course_id, user_id, is_active, is_paid, enrolled_at) VALUES (%s, %s, %s, %s, %s, %s)",
            (str(uuid.uuid4()), course_id, student_id, True, True, now_utc)
        )

        conn.commit()
        print(f"Success! Test course and lesson created with UTC sync.")
        print(f"Course ID: {course_id}")
        print(f"Wait for the 30-minute reminder notification!")

    except Exception as e:
        print(f"Error: {e}")
        conn.rollback()
    finally:
        cur.close()
        conn.close()

if __name__ == "__main__":
    seed_test_lesson()
