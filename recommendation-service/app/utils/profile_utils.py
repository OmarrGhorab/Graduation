from typing import Any, Dict, List


def _first(profile: Dict[str, Any], *keys: str, default=None):
    for key in keys:
        value = profile.get(key)
        if value is not None:
            return value
    return default


def _item_first(item: Dict[str, Any], *keys: str, default=None):
    for key in keys:
        value = item.get(key)
        if value is not None:
            return value
    return default


def list_course_analytics(profile: Dict[str, Any]) -> List[Dict[str, Any]]:
    profile = profile or {}
    return _first(profile, "AllAnalytics", "courseAnalytics", default=[]) or []


def list_subject_preferences(profile: Dict[str, Any]) -> List[Dict[str, Any]]:
    profile = profile or {}
    return _first(profile, "SubjectPreferences", "subjectPreferences", default=[]) or []


def list_preview_interests(profile: Dict[str, Any]) -> List[Dict[str, Any]]:
    profile = profile or {}
    return _first(profile, "PreviewInterests", "previewInterests", default=[]) or []


def list_user_interests(profile: Dict[str, Any]) -> List[str]:
    profile = profile or {}
    return _first(profile, "UserInterests", "userInterests", default=[]) or []


def list_cart_subjects(profile: Dict[str, Any]) -> List[str]:
    profile = profile or {}
    return _first(profile, "CartSubjects", "cartSubjects", default=[]) or []


def get_enrolled_ids_from_profile(profile: Dict[str, Any]) -> List[str]:
    enrolled_ids: List[str] = []
    for item in list_course_analytics(profile):
        if not isinstance(item, dict):
            continue
        course_id = _item_first(item, "CourseID", "courseId")
        if course_id:
            enrolled_ids.append(str(course_id))
    return enrolled_ids


def get_subject_name(item: Dict[str, Any]) -> str:
    if not isinstance(item, dict):
        return ""
    return str(_item_first(item, "SubjectName", "subjectName", default="") or "")


def build_behavior_query(profile: Dict[str, Any]) -> str:
    """Builds a compact retrieval query from watch-time driven behavior."""
    profile = profile or {}
    parts: List[str] = []

    for subject in list_subject_preferences(profile)[:3]:
        name = get_subject_name(subject)
        if name:
            parts.append(f"watched {name}")

    for subject in list_preview_interests(profile)[:3]:
        name = get_subject_name(subject)
        if name:
            parts.append(f"previewed {name}")

    for subject in list_cart_subjects(profile)[:3]:
        if subject:
            parts.append(f"cart interest {subject}")

    for interest in list_user_interests(profile)[:5]:
        if interest:
            parts.append(str(interest))

    watch_patterns = _first(profile, "WatchPatterns", "watchPatterns", default={}) or {}
    completion_tendency = _first(profile, "CompletionTendency", "completionTendency")
    if not completion_tendency and isinstance(watch_patterns, dict):
        completion_tendency = _first(watch_patterns, "CompletionTendency", "completionTendency")
    if completion_tendency:
        parts.append(f"{completion_tendency} completion learner")

    seen = set()
    unique_parts = []
    for part in parts:
        key = part.lower()
        if key in seen:
            continue
        seen.add(key)
        unique_parts.append(part)

    return ", ".join(unique_parts) or "recommended courses"
