from typing import Any, Dict, List

from app.services.course_client import course_client


async def get_user_profile_tool(user_id: str) -> Dict[str, Any]:
    profile = await course_client.get_user_analytics_profile(user_id)
    return {"user_profile": profile or {}}


async def get_user_history_tool(user_id: str) -> Dict[str, Any]:
    profile = await course_client.get_user_analytics_profile(user_id)
    profile = profile or {}

    enrolled_course_ids = []
    for item in profile.get("AllAnalytics", []):
        course_id = item.get("CourseID")
        if course_id:
            enrolled_course_ids.append(str(course_id))

    return {
        "enrolled_course_ids": enrolled_course_ids,
        "cart_subjects": profile.get("CartSubjects", []) or [],
        "interests": profile.get("UserInterests", []) or [],
    }


async def get_recent_cart_activity_tool(user_id: str) -> Dict[str, Any]:
    profile = await course_client.get_user_analytics_profile(user_id)
    profile = profile or {}
    return {"cart_subjects": profile.get("CartSubjects", []) or []}
