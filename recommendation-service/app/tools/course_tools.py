from typing import Any, Dict, List

from app.retrieval.hybrid_search import search_relevant_courses


async def search_relevant_courses_tool(
    user_id: str,
    query: str,
    top_k: int = 20,
    exclude_course_ids: List[str] | None = None,
) -> Dict[str, Any]:
    courses = await search_relevant_courses(
        user_id=user_id,
        query=query,
        top_k=top_k,
        exclude_course_ids=exclude_course_ids or [],
    )
    return {"courses": courses}


async def get_trending_courses_tool(limit: int = 10) -> Dict[str, Any]:
    from app.services.recommendation_engine import get_trending_recommendations

    courses = await get_trending_recommendations()
    return {"courses": (courses or [])[:limit]}
