import uuid
import bcrypt
import psycopg2
from datetime import datetime

# Configuration
DB_URL = "postgresql://graduation:graduation_secret@localhost:5432/graduation"

def get_password_hash(password: str):
    # Hash a password for the first time
    # (bcrypt requires bytes, so we encode the string)
    salt = bcrypt.gensalt()
    hashed = bcrypt.hashpw(password.encode('utf-8'), salt)
    return hashed.decode('utf-8')

def seed_students():
    print("Starting Seeding Process for 4 Students...")
    conn = psycopg2.connect(DB_URL)
    cur = conn.cursor()
    now = datetime.utcnow()
    
    password = "password123"
    hashed_password = get_password_hash(password)
    
    students_to_create = [
        {"name": "Student One", "username": "student1", "email": "student1@example.com"},
        {"name": "Student Two", "username": "student2", "email": "student2@example.com"},
        {"name": "Student Three", "username": "student3", "email": "student3@example.com"},
        {"name": "Student Four", "username": "student4", "email": "student4@example.com"},
    ]

    try:
        for student in students_to_create:
            u_id = str(uuid.uuid4())
            print(f"Creating student: {student['email']}...")
            
            # Check if user already exists
            cur.execute('SELECT id FROM auth."User" WHERE email = %s OR username = %s', (student['email'], student['username']))
            if cur.fetchone():
                print(f"User {student['email']} already exists. Skipping.")
                continue

            cur.execute(
                """INSERT INTO auth.\"User\" (id, name, username, email, password, role, \"onboardingCompleted\", verified, \"createdAt\", \"updatedAt\") 
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)""",
                (u_id, student['name'], student['username'], student['email'], hashed_password, "STUDENT", True, True, now, now)
            )
            
            # Also create a device record so they can login without device verification if needed
            # (Though their login logic might handle first 2 devices as trusted)
            d_id = str(uuid.uuid4())
            cur.execute(
                """INSERT INTO auth.\"UserDevice\" (id, \"userId\", \"deviceFingerprint\", \"deviceName\", \"isTrusted\", \"createdAt\", \"updatedAt\")
                VALUES (%s, %s, %s, %s, %s, %s, %s)""",
                (d_id, u_id, "seed-device-fingerprint-" + u_id[:8], "Seeder Device", True, now, now)
            )

        conn.commit()
        print("\n4 Students Created Successfully!")
        print("-" * 30)
        for student in students_to_create:
            print(f"Email: {student['email']}")
            print(f"Password: {password}")
            print("-" * 30)

    except Exception as e:
        print(f"❌ Error during seeding: {e}")
        conn.rollback()
    finally:
        cur.close()
        conn.close()

if __name__ == "__main__":
    seed_students()
