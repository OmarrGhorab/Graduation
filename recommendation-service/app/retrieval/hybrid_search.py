import hashlib
import json
import logging
from typing import Dict, List, Optional

import redis.asyncio as redis

from app.config import settings
from app.retrieval.embedding_service import embedding_service
from app.retrieval.vector_store import vector_store
from app.services.course_client import course_client
from app.utils.profile_utils import get_enrolled_ids_from_profile

logger = logging.getLogger(__name__)
redis_conn = redis.from_url(settings.REDIS_URL, decode_responses=True)


def _cache_key(user_id: str, query: str, exclude_ids: List[str], filter_enrolled: bool) -> str:
    raw = f"{user_id}:{query}:{','.join(sorted(exclude_ids))}:{filter_enrolled}".encode("utf-8")
    return f"retrieval:v1:{hashlib.sha256(raw).hexdigest()}"


def _get_cluster_affinity(user_id: str) -> Dict[str, float]:
    """Best-effort lookup of the user's cluster course-affinity map.

    Never raises — clustering is an optional enrichment signal, so any failure
    (no assignment, DB error) degrades gracefully to "no cluster boost".
    """
    from app.models.database import SessionLocal
    from app.clustering.cluster_service import ClusterService

    db = SessionLocal()
    try:
        return ClusterService(db).get_course_affinity_for_user(str(user_id))
    except Exception as exc:
        logger.warning(f"Cluster affinity lookup failed for {user_id}: {exc}")
        return {}
    finally:
        db.close()


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
    filter_enrolled: bool = True,
) -> List[Dict]:
    exclude_course_ids = exclude_course_ids or []
    limit = top_k or settings.RETRIEVAL_TOP_K
    cache_key = _cache_key(user_id, query, exclude_course_ids, filter_enrolled)

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
                "teacherName": payload.get("teacherName"),
                "teacherProfileImg": payload.get("teacherProfileImg"),
                "source": ["semantic_similarity"],
            }
        )

    user_profile = {}
    enrolled_ids = set()
    if filter_enrolled:
        try:
            user_profile = await course_client.get_user_analytics_profile(user_id)
        except Exception:
            user_profile = {}
        enrolled_ids = set(get_enrolled_ids_from_profile(user_profile))

    cluster_affinity = _get_cluster_affinity(user_id)

    final = []
    for item in filtered:
        if filter_enrolled and item["courseId"] in enrolled_ids:
            continue
        cluster_score = cluster_affinity.get(item["courseId"], 0.0)
        score = _hybrid_score(
            similarity=float(item.get("similarityScore", 0.0)),
            popularity=float(item.get("enrolledCount", 0) or 0),
            teacher_score=float(item.get("teacherScore", 0) or 0),
            cluster_score=cluster_score,
        )
        item["hybridScore"] = score
        # 0.10 is the cluster weight in _hybrid_score — surface its contribution
        item["clusterContribution"] = round(0.10 * cluster_score, 4)
        if cluster_score > 0 and "cluster_affinity" not in item["source"]:
            item["source"].append("cluster_affinity")
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
    return get_enrolled_ids_from_profile(profile)
