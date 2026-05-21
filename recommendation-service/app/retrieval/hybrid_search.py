import hashlib
import json
import logging
from typing import Dict, List, Optional

import redis.asyncio as redis

from app.config import settings
from app.retrieval.embedding_service import embedding_service
from app.retrieval.vector_store import vector_store
from app.services.course_client import course_client

logger = logging.getLogger(__name__)
redis_conn = redis.from_url(settings.REDIS_URL, decode_responses=True)


def _cache_key(user_id: str, query: str, exclude_ids: List[str]) -> str:
    raw = f"{user_id}:{query}:{','.join(sorted(exclude_ids))}".encode("utf-8")
    return f"retrieval:v1:{hashlib.sha256(raw).hexdigest()}"


def _hybrid_score(similarity: float, popularity: float, teacher_score: float, cluster_score: float = 0.0) -> float:
    return (
        0.60 * similarity
        + 0.20 * min(popularity / 1000.0, 1.0)
        + 0.10 * min(teacher_score / 5.0, 1.0)
        + 0.10 * cluster_score
    )


async def search_relevant_courses(
    user_id: str,
    query: str,
    top_k: Optional[int] = None,
    exclude_course_ids: Optional[List[str]] = None,
) -> List[Dict]:
    exclude_course_ids = exclude_course_ids or []
    limit = top_k or settings.RETRIEVAL_TOP_K
    cache_key = _cache_key(user_id, query, exclude_course_ids)

    try:
        cached = await redis_conn.get(cache_key)
        if cached:
            return json.loads(cached)
    except Exception as exc:
        logger.warning(f"Retrieval cache read failed: {exc}")

    query_vector = await embedding_service.embed_text(query, normalize=True)
    try:
        if query_vector:
            await vector_store.ensure_collections(len(query_vector))
    except Exception as exc:
        logger.warning(f"Collection setup skipped or failed: {exc}")

    results = await vector_store.search_courses(query_vector, top_k=limit * 2)
    filtered = []

    for item in results:
        course_id = str(item.get("courseId"))
        if course_id in exclude_course_ids:
            continue
        payload = item.get("payload") or {}
        filtered.append(
            {
                "courseId": course_id,
                "similarityScore": float(item.get("similarityScore", 0.0)),
                "title": payload.get("title"),
                "subjectName": payload.get("subject"),
                "courseImage": payload.get("courseImage"),
                "price": payload.get("price"),
                "currency": payload.get("currency"),
                "enrolledCount": payload.get("popularity_score", 0),
                "teacherScore": payload.get("teacher_score", 0),
                "source": ["semantic_similarity"],
            }
        )

    try:
        user_profile = await course_client.get_user_analytics_profile(user_id)
    except Exception:
        user_profile = {}

    enrolled_ids = set()
    if user_profile and "AllAnalytics" in user_profile:
        enrolled_ids = {str(a.get("CourseID")) for a in user_profile.get("AllAnalytics", []) if a.get("CourseID")}

    final = []
    for item in filtered:
        if item["courseId"] in enrolled_ids:
            continue
        score = _hybrid_score(
            similarity=float(item.get("similarityScore", 0.0)),
            popularity=float(item.get("enrolledCount", 0) or 0),
            teacher_score=float(item.get("teacherScore", 0) or 0),
        )
        item["hybridScore"] = score
        final.append(item)

    final.sort(key=lambda x: x.get("hybridScore", 0.0), reverse=True)
    final = final[:limit]

    try:
        await redis_conn.setex(cache_key, settings.RETRIEVAL_CACHE_TTL, json.dumps(final))
    except Exception as exc:
        logger.warning(f"Retrieval cache write failed: {exc}")

    return final


async def get_enrolled_course_ids(user_id: str) -> List[str]:
    profile = await course_client.get_user_analytics_profile(user_id)
    if not profile or "AllAnalytics" not in profile:
        return []
    return [str(a.get("CourseID")) for a in profile.get("AllAnalytics", []) if a.get("CourseID")]
