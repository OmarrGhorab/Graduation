import uuid
import random
import bcrypt
import psycopg2
from psycopg2.extras import execute_values
from datetime import datetime, timedelta

# Configuration
DB_URL = "postgresql://graduation:graduation_secret@localhost:5432/graduation"

def get_password_hash(password: str):
    salt = bcrypt.gensalt()
    hashed = bcrypt.hashpw(password.encode('utf-8'), salt)
    return hashed.decode('utf-8')

def reseed():
    print("Starting FULL Reseeding Process...")
    conn = psycopg2.connect(DB_URL)
    cur = conn.cursor()
    now = datetime.utcnow()
    
    try:
        # 1. TRUNCATE TABLES (Only those that exist)
        print("Cleaning all databases (auth & public)...")
        tables_to_clean = [
            'auth."User"', 'auth."Interest"', 'auth."UserInterest"', 'auth."ParentChildLink"',
            'auth."Session"', 'auth."ParentLinkRequest"',
            'public.courses', 'public.subjects', 'public.lessons', 'public.enrollments', 
            'public.user_course_analytics', 'public.user_lesson_progress'
        ]
        
        for table in tables_to_clean:
            try:
                cur.execute(f"TRUNCATE {table} CASCADE;")
            except Exception as e:
                cur.execute("ROLLBACK")
                print(f"   - Skipping {table} (might not exist yet)")
                continue

        # 2. SEED SUBJECTS & INTERESTS
        print("Seeding Subjects & Interests...")
        subjects = [
            (str(uuid.uuid4()), "AI & Machine Learning", "Neural networks and deep learning", "brain"),
            (str(uuid.uuid4()), "Fullstack Development", "Modern web technologies", "code"),
            (str(uuid.uuid4()), "Cloud Computing", "AWS, Azure and DevOps", "cloud"),
            (str(uuid.uuid4()), "Security", "Ethical hacking and defense", "shield"),
            (str(uuid.uuid4()), "Data Science", "Statistics and Analytics", "bar-chart")
        ]
        execute_values(cur, "INSERT INTO public.subjects (id, name, description, icon) VALUES %s", subjects)
        
        interests = [(s[0], s[1], now, now) for s in subjects]
        execute_values(cur, "INSERT INTO auth.\"Interest\" (id, name, \"createdAt\", \"updatedAt\") VALUES %s", interests)

        # 3. CREATE USERS
        print("Creating 4 benchmark users...")
        pwd = get_password_hash("password123")
        
        # Teacher
        teacher_id = str(uuid.uuid4())
        cur.execute(
            """INSERT INTO auth.\"User\" (id, name, username, email, password, role, \"onboardingCompleted\", verified, \"createdAt\", \"updatedAt\") 
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)""",
            (teacher_id, "Prof. Omar Ghorab", "teacher_omar", "teacher@example.com", pwd, "TEACHER", True, True, now, now)
        )
        
        # Student (The Child)
        student_id = str(uuid.uuid4())
        cur.execute(
            """INSERT INTO auth.\"User\" (id, name, username, email, password, role, \"onboardingCompleted\", verified, \"createdAt\", \"updatedAt\") 
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)""",
            (student_id, "Adham Student", "student_adham", "student@example.com", pwd, "STUDENT", True, True, now, now)
        )
        
        # Parent
        parent_id = str(uuid.uuid4())
        cur.execute(
            """INSERT INTO auth.\"User\" (id, name, username, email, password, role, \"onboardingCompleted\", verified, \"createdAt\", \"updatedAt\") 
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)""",
            (parent_id, "Walid Parent", "parent_walid", "parent@example.com", pwd, "PARENT", True, True, now, now)
        )
        
        # Linking Parent and Student
        cur.execute(
            "INSERT INTO auth.\"ParentChildLink\" (id, \"parentId\", \"childId\", \"createdAt\") VALUES (%s, %s, %s, %s)",
            (str(uuid.uuid4()), parent_id, student_id, now)
        )
        
        # Admin
        admin_id = str(uuid.uuid4())
        cur.execute(
            """INSERT INTO auth.\"User\" (id, name, username, email, password, role, \"onboardingCompleted\", verified, \"createdAt\", \"updatedAt\") 
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)""",
            (admin_id, "System Admin", "admin_sys", "admin@example.com", pwd, "HR", True, True, now, now)
        )

        # 4. CREATE 10 COURSES
        print("Creating 10 courses with detailed lessons...")
        
        course_ids = []
        for i in range(10):
            is_paid = i < 5 # First 5 are paid, last 5 are free
            billing = "MONTHLY" if i % 2 == 0 else "ONE_TIME"
            # Use fixed ID for the first course to accommodate user requests
            c_id = "4cc33118-e390-48da-afbc-220ec38925a2" if i == 0 else str(uuid.uuid4())
            course_ids.append(c_id)
            
            title = f"{subj[1]} Masterclass {i+1}" if is_paid else f"Intro to {subj[1]} {i+1}"
            
            cur.execute("""
                INSERT INTO public.courses (id, title, description, "subject_id", "teacher_id", "delivery_type", price, status, "is_paid", "free_trial_lessons", "total_lessons", "billing_type", "created_at", "updated_at") 
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            """, (
                c_id, title, f"Full depth guidance into {subj[1]}.",
                subj[0], teacher_id, "ONLINE", 250.0 if is_paid else 0.0, "ACTIVE", is_paid, 2 if is_paid else 0, 5, billing, now, now
            ))
            
            # Create 5 lessons for each course
            for l_idx in range(1, 6):
                l_id = str(uuid.uuid4())
                # For the specific course 4cc33, make ALL lessons free. 
                # For others, keep the "first 2 free" logic for paid courses.
                if c_id == "4cc33118-e390-48da-afbc-220ec38925a2":
                    is_free = True
                else:
                    is_free = (not is_paid) or (l_idx <= 2)
                
                # Mock video and material
                video_url = "https://res.cloudinary.com/demo/video/upload/dog.mp4" 
                material_url = "https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf" 
                
                cur.execute("""
                    INSERT INTO public.lessons (id, course_id, title, description, lesson_number, scheduled_at, status, is_free, video_url, materials_url, duration, delivery_type)
                    VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                """, (
                    l_id, c_id, f"Lesson {l_idx}: Mastery Path", f"Advanced module covering {title} details.", 
                    l_idx, now + timedelta(days=l_idx), 'SCHEDULED', 
                    is_free, video_url, material_url, 30, 'ONLINE'
                ))

        conn.commit()
        print("\nDatabase Restored Successfully!")
        print(f"   - Teacher: teacher@example.com / password123")
        print(f"   - Student: student@example.com / password123")
        print(f"   - Parent:  parent@example.com  / password123")
        print(f"   - Admin:   admin@example.com   / password123")
        print(f"   - Courses: 10 total (5 Paid, 5 Free) | Lessons: 5 each")
        print(f"   - Billing: Mixed (ONE_TIME & MONTHLY)")

    except Exception as e:
        print(f"Error: {e}")
        conn.rollback()
    finally:
        cur.close()
        conn.close()

if __name__ == "__main__":
    reseed()
