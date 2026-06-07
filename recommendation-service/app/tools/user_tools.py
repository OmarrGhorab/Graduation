from typing import Any, Dict, List

from app.services.course_client import course_client
from app.utils.profile_utils import (
    build_behavior_query,
    get_enrolled_ids_from_profile,
    list_cart_subjects,
    list_subject_preferences,
    list_user_interests,
)


async def get_user_profile_tool(user_id: str) -> Dict[str, Any]:
    profile = await course_client.get_user_analytics_profile(user_id)
    return {"user_profile": profile or {}}


async def get_user_history_tool(user_id: str) -> Dict[str, Any]:
    profile = await course_client.get_user_analytics_profile(user_id)
    profile = profile or {}

    return {
        "enrolled_course_ids": get_enrolled_ids_from_profile(profile),
        "cart_subjects": list_cart_subjects(profile),
        "interests": list_user_interests(profile),
        "subject_preferences": list_subject_preferences(profile),
        "behavior_query": build_behavior_query(profile),
    }


async def get_recent_cart_activity_tool(user_id: str) -> Dict[str, Any]:
    profile = await course_client.get_user_analytics_profile(user_id)
    profile = profile or {}
    return {"cart_subjects": list_cart_subjects(profile)}
