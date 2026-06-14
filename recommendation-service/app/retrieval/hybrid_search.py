import hashlib
import json
import logging
from typing import Any, Dict, List, Optional

import redis.asyncio as redis

from app.config import settings
from app.retrieval.embedding_service import embedding_service
from app.retrieval.vector_store import vector_store
from app.services.course_client import course_client
from app.utils.profile_utils import (
    get_enrolled_ids_from_profile,
    get_subject_name,
    list_cart_subjects,
    list_preview_interests,
    list_subject_preferences,
    list_user_interests,
)

logger = logging.getLogger(__name__)
redis_conn = redis.from_url(settings.REDIS_URL, decode_responses=True)


def _normalize_text(value: Any) -> str:
    return str(value or "").strip().lower()


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


def _profile_subject_boost(course_subject: str, user_profile: Dict) -> float:
    if not course_subject:
        return 0.0
    watched_subjects = {
        get_subject_name(item).lower()
        for item in list_subject_preferences(user_profile)
        if get_subject_name(item)
    }
    if course_subject.lower() in watched_subjects:
        return 0.08
    return 0.0


def _profile_interest_terms(user_profile: Dict) -> set[str]:
    terms: set[str] = set()

    for interest in list_user_interests(user_profile):
        normalized = _normalize_text(interest)
        if normalized:
            terms.add(normalized)

    for subject in list_cart_subjects(user_profile):
        normalized = _normalize_text(subject)
        if normalized:
            terms.add(normalized)

    for preview in list_preview_interests(user_profile):
        subject_name = get_subject_name(preview)
        normalized = _normalize_text(subject_name)
        if normalized:
            terms.add(normalized)

    for subject in list_subject_preferences(user_profile):
        subject_name = get_subject_name(subject)
        normalized = _normalize_text(subject_name)
        if normalized:
            terms.add(normalized)

    return terms


def _interest_match_score(course: Dict[str, Any], user_profile: Dict) -> float:
    terms = _profile_interest_terms(user_profile)
    if not terms:
        return 0.0

    haystack = " ".join(
        filter(
            None,
            [
                _normalize_text(course.get("title")),
                _normalize_text(course.get("description")),
                _normalize_text(course.get("subjectName")),
                _normalize_text(course.get("teacherName")),
            ],
        )
    )
    if not haystack:
        return 0.0

    score = 0.0
    for term in terms:
        if term in haystack:
            score += 0.18
        else:
            term_tokens = [token for token in term.split() if len(token) >= 3]
            if term_tokens and all(token in haystack for token in term_tokens):
                score += 0.12
            elif any(token in haystack for token in term_tokens):
                score += 0.06

    return min(score, 0.65)


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
        if not query_vector:
            return []
    except Exception as exc:
        logger.warning(f"Query vector generation produced no usable vector: {exc}")
        return []

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

    user_profile = course_client.get_cached_user_analytics_profile(user_id)
    enrolled_ids = set()
    if filter_enrolled:
        enrolled_ids = set(get_enrolled_ids_from_profile(user_profile))

    cluster_affinity = _get_cluster_affinity(user_id)
    interest_terms = _profile_interest_terms(user_profile)
    # Cold-start: if user has no enrollments, amplify interest signal
    is_cold_start = len(enrolled_ids) == 0

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
        subject_boost = _profile_subject_boost(str(item.get("subjectName") or ""), user_profile)
        interest_boost = _interest_match_score(item, user_profile)
        if interest_boost == 0.0 and is_cold_start and interest_terms:
            similarity = float(item.get("similarityScore", 0.0))
            if similarity >= 0.50:
                interest_boost = (similarity - 0.45) * 0.6
        # For cold-start users, interests are the only personalisation signal
        # so we amplify them to differentiate recommendations across users
        if is_cold_start and interest_boost > 0:
            interest_boost *= 2.5
        score += subject_boost
        score += interest_boost
        item["hybridScore"] = score
        item["profileSubjectBoost"] = round(subject_boost, 4)
        item["interestBoost"] = round(interest_boost, 4)
        # 0.10 is the cluster weight in _hybrid_score — surface its contribution
        item["clusterContribution"] = round(0.10 * cluster_score, 4)
        if cluster_score > 0 and "cluster_affinity" not in item["source"]:
            item["source"].append("cluster_affinity")
        if interest_terms and interest_boost > 0 and "onboarding_interest" not in item["source"]:
            item["source"].append("onboarding_interest")
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
