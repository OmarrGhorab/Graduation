import psycopg2
import sys

def update_lessons():
    try:
        conn = psycopg2.connect("postgresql://graduation:graduation_secret@localhost:5432/graduation")
        cur = conn.cursor()
        
        # Update all lessons for the specific course to be ONLINE and FREE
        course_id = '4cc33118-e390-48da-afbc-220ec38925a2'
        cur.execute("""
            UPDATE public.lessons 
            SET delivery_type = 'ONLINE', 
                is_free = true,
                duration = 30 
            WHERE course_id = %s
        """, (course_id,))
        
        conn.commit()
        print(f"Successfully updated lessons for course {course_id} to ONLINE and FREE (30s duration)")
        cur.close()
        conn.close()
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    update_lessons()
