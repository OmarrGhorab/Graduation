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

        # 3.5. ASSIGN INTERESTS TO STUDENT (For Recommendations)
        print("Assigning 3 interests to student...")
        student_interests = []
        selected_interests = random.sample(subjects, 3)
        for si in selected_interests:
            student_interests.append((student_id, si[0], now))
        
        execute_values(cur, "INSERT INTO auth.\"UserInterest\" (\"userId\", \"interestId\", \"createdAt\") VALUES %s", student_interests)
        
        # Admin
        admin_id = str(uuid.uuid4())
        cur.execute(
            """INSERT INTO auth.\"User\" (id, name, username, email, password, role, \"onboardingCompleted\", verified, \"createdAt\", \"updatedAt\") 
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)""",
            (admin_id, "System Admin", "admin_sys", "admin@example.com", pwd, "HR", True, True, now, now)
        )

        # 4. CREATE 15 COURSES
        print("Creating 15 courses with detailed lessons...")
        
        VALID_THUMBS = [
            "https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=800&q=80",
            "https://images.unsplash.com/photo-1501504905252-473c47e087f8?w=800&q=80",
            "https://images.unsplash.com/photo-1498050108023-c5249f4df085?w=800&q=80",
            "https://images.unsplash.com/photo-1504639725590-34d0984388bd?w=800&q=80",
            "https://images.unsplash.com/photo-1517694712202-14dd9538aa97?w=800&q=80"
        ]
        VALID_HLS_VIDEOS = [
            ("https://res.cloudinary.com/demo/video/upload/sp_auto/elephants.m3u8", "demo/elephants"),
            ("https://res.cloudinary.com/demo/video/upload/sp_auto/sea_turtle.m3u8", "demo/sea_turtle")
        ]
        
        course_prefixes = ["Mastering", "Intro to", "Advanced", "The Art of", "Modern", "Professional", "Complete", "Deep Dive:", "Essential"]
        course_topics = ["React Native", "Docker", "Python", "Kubernetes", "UI/UX", "Machine Learning", "Cloud Architecture", "Go Programming", "Cybersecurity", "Blockchain", "Data Engineering", "Next.js"]
        course_suffixes = ["for Experts", "Masterclass", "Bootcamp", "Essentials", "Simplified", "Course 2024", "Handbook", "Workshop"]

        course_ids = []
        for i in range(15):
            # Variety Coverage
            is_paid = (i % 5 != 0) 
            delivery = "ONLINE" if i % 2 == 0 else "OFFLINE"
            billing = "MONTHLY" if i % 3 == 0 else "ONE_TIME"
            subj = subjects[i % len(subjects)]
            total_less = random.randint(4, 20)
            
            prefix = random.choice(course_prefixes)
            topic = random.choice(course_topics) if i > 0 else subj[1]
            suffix = random.choice(course_suffixes)
            title = f"{prefix} {topic} {suffix}" if i > 0 else f"{subj[1]} Masterclass"
            
            c_id = "4cc33118-e390-48da-afbc-220ec38925a2" if i == 0 else str(uuid.uuid4())
            course_ids.append((c_id, title, delivery, total_less))
            
            thumb = random.choice(VALID_THUMBS)
            v_url, v_id = random.choice(VALID_HLS_VIDEOS)
            
            cur.execute("""
                INSERT INTO public.courses (
                    id, title, description, "subject_id", "teacher_id", "delivery_type", 
                    price, status, "is_paid", "free_trial_lessons", "total_lessons", 
                    "billing_type", "course_image", "preview_video_url", "preview_video_public_id",
                    "created_at", "updated_at"
                ) 
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            """, (
                c_id, title, f"Full depth guidance into {subj[1]}.",
                subj[0], teacher_id, delivery, 250.0 if is_paid else 0.0, "ACTIVE", is_paid, 2 if is_paid else 0, total_less, billing,
                thumb, v_url, v_id, now, now
            ))
            
            # Create lessons for each course
            for l_idx in range(1, total_less + 1):
                l_id = str(uuid.uuid4())
                if c_id == "4cc33118-e390-48da-afbc-220ec38925a2":
                    is_free = True
                else:
                    is_free = (not is_paid) or (l_idx <= 2)
                
                # Mock video and material
                video_url = "https://res.cloudinary.com/demo/video/upload/dog.mp4" if delivery == "ONLINE" else None
                material_url = "https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf" if delivery == "ONLINE" else None
                
                cur.execute("""
                    INSERT INTO public.lessons (id, course_id, title, description, lesson_number, scheduled_at, status, is_free, video_url, materials_url, duration_minutes, delivery_type)
                    VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                """, (
                    l_id, c_id, f"Lesson {l_idx}: Mastery Path", f"Advanced module covering {title} details.", 
                    l_idx, now + timedelta(days=l_idx), 'SCHEDULED', 
                    is_free, video_url, material_url, 30, delivery
                ))

        # 5. ENROLL STUDENT IN SOME COURSES (Especially for testing)
        print("Enrolling student in first 10 courses...")
        for i in range(10):
            c_id = course_ids[i][0]
            cur.execute("""
                INSERT INTO public.enrollments (id, user_id, course_id, is_active, enrolled_at, updated_at)
                VALUES (%s, %s, %s, %s, %s, %s)
            """, (
                str(uuid.uuid4()), student_id, c_id, True, now, now
            ))

        conn.commit()
        print("\nDatabase Restored Successfully!")
        print(f"   - Teacher: teacher@example.com / password123")
        print(f"   - Student: student@example.com / password123")
        print(f"   - Parent:  parent@example.com  / password123")
        print(f"   - Admin:   admin@example.com   / password123")
        print(f"   - Courses: 15 total | Lessons: Dynamic (4-20 each)")
        print(f"   - Features: Thumbnails, Previews, All Billing/Delivery types")

    except Exception as e:
        print(f"Error: {e}")
        conn.rollback()
    finally:
        cur.close()
        conn.close()

if __name__ == "__main__":
    reseed()
