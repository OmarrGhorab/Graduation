from typing import Dict, List


def build_course_embedding_text(course: Dict) -> str:
    subject = course.get("subject", {})
    subject_name = subject.get("name") if isinstance(subject, dict) else None
    subject_name = subject_name or course.get("subjectName", "Unknown")
    categories = course.get("categories") or course.get("tags") or []
    if isinstance(categories, list):
        categories_text = ", ".join([str(x) for x in categories if x])
    else:
        categories_text = str(categories)

    parts = [
        f"Title: {course.get('title', '')}",
        f"Subject: {subject_name}",
        f"Categories: {categories_text}",
        f"Description: {course.get('description', '')}",
        f"Teacher: {course.get('teacherName', '')}",
        f"Popularity: {course.get('enrollmentCount', 0)}",
    ]
    return "\n".join([p for p in parts if p.strip()])


def build_course_embedding_payload(course: Dict) -> Dict:
    subject = course.get("subject", {})
    subject_name = subject.get("name") if isinstance(subject, dict) else None
    subject_name = subject_name or course.get("subjectName")
    categories = course.get("categories") or course.get("tags") or []
    if not isinstance(categories, list):
        categories = [str(categories)] if categories else []

    return {
        "title": course.get("title"),
        "subject": subject_name,
        "categories": categories,
        "courseImage": course.get("courseImage"),
        "price": course.get("price"),
        "currency": course.get("currency"),
        "popularity_score": course.get("enrollmentCount", 0),
        "teacher_score": course.get("teacherAuthority", 0),
        "teacherName": course.get("teacherName"),
        "teacherProfileImg": course.get("teacherProfileImg"),
        "updated_at": course.get("updatedAt"),
    }
