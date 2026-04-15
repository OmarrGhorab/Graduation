import uuid
import random
from datetime import datetime, timedelta
from faker import Faker
import psycopg2
from psycopg2.extras import execute_values
import bcrypt

# Configuration
DB_URL = "postgresql://graduation:graduation_secret@localhost:5432/graduation"
fake = Faker()

def get_password_hash(password: str):
    # Hash a password for the first time
    # (bcrypt requires bytes, so we encode the string)
    salt = bcrypt.gensalt()
    hashed = bcrypt.hashpw(password.encode('utf-8'), salt)
    return hashed.decode('utf-8')

def seed():
    print("🚀 Starting Seeding Process...")
    conn = psycopg2.connect(DB_URL)
    cur = conn.cursor()

    try:
        # 0. Flush Redis Cache (via Docker)
        print("🧹 Flushing Redis cache...")
        import subprocess
        try:
            subprocess.run(["docker", "exec", "graduation-redis-1", "redis-cli", "FLUSHALL"], capture_output=True)
            print("✨ Redis cleaned!")
        except Exception as e:
            print(f"⚠️ Redis flush skipped (is Docker running?): {e}")

        # 1. Clean existing data (optional, but good for fresh test)
        # Note: Order is important due to foreign keys
        print("🧹 Cleaning existing data...")
        cur.execute("TRUNCATE auth.\"User\", auth.\"Interest\", auth.\"UserInterest\", public.courses, public.subjects, public.enrollments, public.user_course_analytics, public.user_lesson_progress CASCADE;")

        # 2. Seed Subjects
        print("📚 Seeding Subjects...")
        subjects = [
            (str(uuid.uuid4()), "Mathematics", "Advanced and Basic Math", "pi"),
            (str(uuid.uuid4()), "Physics", "Science & Mechanics", "atom"),
            (str(uuid.uuid4()), "Programming", "Python, Go, JS", "code"),
            (str(uuid.uuid4()), "Art", "Design and Painting", "palette"),
            (str(uuid.uuid4()), "Languages", "English and Arabic", "globe")
        ]
        execute_values(cur, "INSERT INTO public.subjects (id, name, description, icon) VALUES %s", subjects)
        
        # 3. Seed Interests (linked to Subjects)
        print("🏷️ Seeding Auth Interests...")
        now = datetime.now()
        interests = [(s[0], s[1], now, now) for s in subjects]
        execute_values(cur, "INSERT INTO auth.\"Interest\" (id, name, \"createdAt\", \"updatedAt\") VALUES %s", interests)

        # 4. Create Teachers
        print("👩‍🏫 Seeding Teachers...")
        teacher_id = str(uuid.uuid4())
        cur.execute(
            """INSERT INTO auth.\"User\" (id, name, username, email, password, role, \"onboardingCompleted\", verified, \"createdAt\", \"updatedAt\") 
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)""",
            (teacher_id, "Dr. Ahmed Zewail", "ahmed_zewail", "teacher@example.com", get_password_hash("password123"), "TEACHER", True, True, now, now)
        )

        # 5. Create Courses
        print("🎓 Seeding Courses...")
        course_data = []
        for s_id, s_name, _, _ in subjects:
            for i in range(2):
                c_id = str(uuid.uuid4())
                course_data.append((
                    c_id, f"{s_name} {i+1}01", f"A comprehensive course on {s_name}.",
                    s_id, teacher_id, "ONLINE", 100.0, "ACTIVE", True, now, now
                ))
        execute_values(cur, """
            INSERT INTO public.courses (id, title, description, "subject_id", "teacher_id", "delivery_type", price, status, "is_paid", "created_at", "updated_at") 
            VALUES %s
        """, course_data)

        # 6. Create Students & Profiles
        print("🧑‍🎓 Seeding Students & Engagement...")
        students = []
        for i in range(5):
            u_id = str(uuid.uuid4())
            students.append(u_id)
            cur.execute(
                """INSERT INTO auth.\"User\" (id, name, username, email, password, role, \"onboardingCompleted\", verified, \"createdAt\", \"updatedAt\") 
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)""",
                (u_id, f"Student {i+1}", f"student_{i+1}", f"student{i+1}@example.com", get_password_hash("password123"), "STUDENT", True, True, now, now)
            )
            
            # Add Interests for Student
            target_interests = random.sample(interests, 2)
            for int_id, _, _, _ in target_interests:
                cur.execute("INSERT INTO auth.\"UserInterest\" (\"userId\", \"interestId\", \"createdAt\") VALUES (%s, %s, %s)", (u_id, int_id, now))

        # 7. Create Mock Analytics for Student 1 (The "Target" for testing)
        target_student = students[0]
        print(f"📈 Creating rich analytics for primary test student: {target_student}")
        
        # Pick a course from 'Mathematics' subject
        math_subject_id = [s[0] for s in subjects if s[1] == "Mathematics"][0]
        math_course_id = [c[0] for c in course_data if c[3] == math_subject_id][0]
        
        # Insert high engagement for math
        cur.execute("""
            INSERT INTO public.user_course_analytics 
            (id, "course_id", "user_id", "total_watch_time", "lessons_completed", "total_lessons", "completion_pct", "engagement_score")
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
        """, (str(uuid.uuid4()), math_course_id, target_student, 5400, 8, 10, 80.0, 92.5))

        conn.commit()
        print("✅ Seeding Completed Successfully!")

    except Exception as e:
        print(f"❌ Error during seeding: {e}")
        conn.rollback()
    finally:
        cur.close()
        conn.close()

if __name__ == "__main__":
    seed()
