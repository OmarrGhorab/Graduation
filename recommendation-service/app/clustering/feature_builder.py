from collections import defaultdict
from typing import Dict, List


def build_user_behavior_summary(user_id: str, user_profile: Dict) -> str:
    user_profile = user_profile or {}
    subject_preferences = user_profile.get("SubjectPreferences") or []
    preview_interests = user_profile.get("PreviewInterests") or []
    cart_subjects = user_profile.get("CartSubjects") or []
    user_interests = user_profile.get("UserInterests") or []
    completion_tendency = user_profile.get("CompletionTendency") or "UNKNOWN"
    avg_completion_pct = user_profile.get("AvgCompletionPct") or 0
    avg_session_duration = user_profile.get("AvgSessionDuration") or 0

    subject_names = [s.get("SubjectName", "") for s in subject_preferences if isinstance(s, dict)]
    preview_subject_names = [p.get("SubjectName", "") for p in preview_interests if isinstance(p, dict)]

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
    all_analytics = user_profile.get("AllAnalytics") or []
    completion_values: List[float] = []
    engagement_values: List[float] = []
    watch_time_total = 0.0
    courses_seen = set()
    category_counts = defaultdict(int)

    for item in all_analytics:
        if not isinstance(item, dict):
            continue
        course_id = item.get("CourseID")
        if course_id:
            courses_seen.add(str(course_id))
        watch_time_total += float(item.get("TotalWatchTime", 0) or 0)
        completion_values.append(float(item.get("CompletionPct", 0) or 0))
        engagement_values.append(float(item.get("EngagementScore", 0) or 0))
        subject_name = item.get("SubjectName")
        if subject_name:
            category_counts[str(subject_name)] += 1

    completion_avg = sum(completion_values) / len(completion_values) if completion_values else 0.0
    engagement_avg = sum(engagement_values) / len(engagement_values) if engagement_values else 0.0

    return {
        "courses_count": float(len(courses_seen)),
        "watch_time_total": watch_time_total,
        "completion_avg": completion_avg,
        "engagement_avg": engagement_avg,
        "cart_subjects_count": float(len(user_profile.get("CartSubjects") or [])),
        "top_category_count": float(max(category_counts.values()) if category_counts else 0),
    }
