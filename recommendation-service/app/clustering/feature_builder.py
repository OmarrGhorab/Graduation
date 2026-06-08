from collections import defaultdict
from typing import Dict, List
from app.utils.profile_utils import (
    get_subject_name,
    list_cart_subjects,
    list_course_analytics,
    list_preview_interests,
    list_subject_preferences,
    list_user_interests,
)


def build_user_behavior_summary(user_id: str, user_profile: Dict) -> str:
    user_profile = user_profile or {}
    subject_preferences = list_subject_preferences(user_profile)
    preview_interests = list_preview_interests(user_profile)
    cart_subjects = list_cart_subjects(user_profile)
    user_interests = list_user_interests(user_profile)
    completion_tendency = user_profile.get("CompletionTendency") or user_profile.get("completionTendency") or "UNKNOWN"
    avg_completion_pct = user_profile.get("AvgCompletionPct") or user_profile.get("avgCompletionPct") or 0
    avg_session_duration = user_profile.get("AvgSessionDuration") or user_profile.get("avgSessionDuration") or 0

    subject_names = [get_subject_name(s) for s in subject_preferences if isinstance(s, dict)]
    preview_subject_names = [get_subject_name(p) for p in preview_interests if isinstance(p, dict)]

    lines = [
        f"User ID: {user_id}",
        f"Interests: {', '.join([str(x) for x in user_interests if x])}",
        f"Most watched subjects: {', '.join([x for x in subject_names if x])}",
        f"Preview interests: {', '.join([x for x in preview_subject_names if x])}",
        f"Cart subjects: {', '.join([str(x) for x in cart_subjects if x])}",
        f"Completion tendency: {completion_tendency}",
        f"Average completion percent: {avg_completion_pct}",
        f"Average session duration: {avg_session_duration}",
    ]
    return "\n".join(lines)


def build_user_numeric_features(user_profile: Dict) -> Dict[str, float]:
    user_profile = user_profile or {}
    all_analytics = list_course_analytics(user_profile)
    completion_values: List[float] = []
    engagement_values: List[float] = []
    watch_time_total = 0.0
    courses_seen = set()
    category_counts = defaultdict(int)

    for item in all_analytics:
        if not isinstance(item, dict):
            continue
        course_id = item.get("CourseID") or item.get("courseId")
        if course_id:
            courses_seen.add(str(course_id))
        watch_time_total += float(item.get("TotalWatchTime", item.get("totalWatchTime", 0)) or 0)
        completion_values.append(float(item.get("CompletionPct", item.get("completionPct", 0)) or 0))
        engagement_values.append(float(item.get("EngagementScore", item.get("engagementScore", 0)) or 0))
        subject_name = item.get("SubjectName") or item.get("subjectName")
        if subject_name:
            category_counts[str(subject_name)] += 1

    completion_avg = sum(completion_values) / len(completion_values) if completion_values else 0.0
    engagement_avg = sum(engagement_values) / len(engagement_values) if engagement_values else 0.0

    return {
        "courses_count": float(len(courses_seen)),
        "watch_time_total": watch_time_total,
        "completion_avg": completion_avg,
        "engagement_avg": engagement_avg,
        "cart_subjects_count": float(len(list_cart_subjects(user_profile))),
        "top_category_count": float(max(category_counts.values()) if category_counts else 0),
    }
