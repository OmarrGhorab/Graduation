import hashlib
import json
import logging
from difflib import SequenceMatcher
from typing import Any, Dict, List, Optional, Tuple

from app.clustering.cluster_service import ClusterService
from app.config import settings
from app.models.database import SessionLocal
from app.retrieval.embedding_service import embedding_service
from app.retrieval.hybrid_search import redis_conn
from app.retrieval.vector_store import vector_store
from app.services.course_client import course_client
from app.utils.profile_utils import get_subject_name, list_subject_preferences

logger = logging.getLogger(__name__)

_DEFAULT_PAGE = 1
_DEFAULT_LIMIT = 10
_MAX_LIMIT = 100
_DEFAULT_AUTOCOMPLETE_LIMIT = 8
_MAX_AUTOCOMPLETE_LIMIT = 20
_MIN_AUTOCOMPLETE_SEARCH_LENGTH = 2
_SEMANTIC_MIN_SCORE = 0.15
_SHORT_QUERY_LEN = 4


def _safe_float(value: Any, default: float = 0.0) -> float:
    try:
        if value is None:
            return default
        return float(value)
    except (TypeError, ValueError):
        return default


def _safe_int(value: Any, default: int = 0) -> int:
    try:
        if value is None:
            return default
        return int(value)
    except (TypeError, ValueError):
        return default


def _normalize_text(value: Any) -> str:
    return str(value or "").strip().lower()


def _search_cache_key(
    user_id: str,
    search: str,
    filters: Dict[str, Any],
    page: int,
    limit: int,
) -> str:
    payload = {
        "user_id": user_id,
        "search": search.strip(),
        "filters": filters,
        "page": page,
        "limit": limit,
    }
    raw = json.dumps(payload, sort_keys=True, default=str).encode("utf-8")
    return f"course-search:v1:{hashlib.sha256(raw).hexdigest()}"


def _autocomplete_cache_key(user_id: str, search: str, limit: int) -> str:
    raw = json.dumps(
        {"user_id": user_id, "search": search.strip(), "limit": limit},
        sort_keys=True,
    ).encode("utf-8")
    return f"course-autocomplete:v1:{hashlib.sha256(raw).hexdigest()}"


def _extract_subject_preferences(user_profile: Dict[str, Any]) -> set[str]:
    return {
        get_subject_name(item).strip().lower()
        for item in list_subject_preferences(user_profile)
        if get_subject_name(item)
    }


def _get_cluster_affinity(user_id: str) -> Dict[str, float]:
    db = SessionLocal()
    try:
        return ClusterService(db).get_course_affinity_for_user(str(user_id))
    except Exception as exc:
        logger.warning(f"Course search cluster affinity lookup failed for {user_id}: {exc}")
        return {}
    finally:
        db.close()


def _course_catalog_index(courses: List[Dict[str, Any]]) -> Dict[str, Dict[str, Any]]:
    indexed: Dict[str, Dict[str, Any]] = {}
    for course in courses:
        course_id = course.get("id") or course.get("courseId")
        if course_id is not None:
            indexed[str(course_id)] = course
    return indexed


def _matches_filters(course: Dict[str, Any], filters: Dict[str, Any]) -> bool:
    subject_id = filters.get("subjectId")
    if subject_id and str(course.get("subjectId")) != str(subject_id):
        return False

    subject_name = _normalize_text(filters.get("subjectName"))
    if subject_name and subject_name not in _normalize_text(course.get("subjectName")):
        return False

    teacher_id = filters.get("teacherId")
    if teacher_id and str(course.get("teacherId")) != str(teacher_id):
        return False

    delivery_type = _normalize_text(filters.get("deliveryType"))
    if delivery_type and _normalize_text(course.get("deliveryType")) != delivery_type:
        return False

    if filters.get("isPaid") is not None:
        course_is_paid = bool(course.get("isPaid"))
        if course_is_paid != bool(filters.get("isPaid")):
            return False

    billing_type = _normalize_text(filters.get("billingType"))
    if billing_type and _normalize_text(course.get("billingType")) != billing_type:
        return False

    status = _normalize_text(filters.get("status"))
    if status and _normalize_text(course.get("status")) != status:
        return False

    min_price = filters.get("minPrice")
    if min_price is not None and _safe_float(course.get("price")) < _safe_float(min_price):
        return False

    max_price = filters.get("maxPrice")
    if max_price is not None and _safe_float(course.get("price")) > _safe_float(max_price):
        return False

    return True


def _keyword_match_score(course: Dict[str, Any], search: str) -> float:
    query = _normalize_text(search)
    if not query:
        return 0.0

    title = _normalize_text(course.get("title"))
    description = _normalize_text(course.get("description"))
    subject_name = _normalize_text(course.get("subjectName"))
    teacher_name = _normalize_text(course.get("teacherName"))

    score = 0.0
    if query in title:
        score += 1.0
    if query in subject_name:
        score += 0.7
    if query in teacher_name:
        score += 0.5
    if query in description:
        score += 0.35

    query_terms = [term for term in query.split() if term]
    searchable = f"{title} {subject_name} {teacher_name} {description}"
    for term in query_terms:
        if term in searchable:
            score += 0.15
    return score


def _tokenize(value: Any) -> List[str]:
    normalized = _normalize_text(value)
    if not normalized:
        return []
    cleaned = normalized.replace("-", " ").replace("/", " ")
    return [token for token in cleaned.split() if token]


def _fuzzy_similarity(left: str, right: str) -> float:
    if not left or not right:
        return 0.0
    return SequenceMatcher(None, left, right).ratio()


def _autocomplete_lexical_score(course: Dict[str, Any], search: str) -> Tuple[float, str]:
    query = _normalize_text(search)
    if not query:
        return 0.0, "none"

    title = _normalize_text(course.get("title"))
    subject_name = _normalize_text(course.get("subjectName"))
    teacher_name = _normalize_text(course.get("teacherName"))
    description = _normalize_text(course.get("description"))

    title_tokens = _tokenize(title)
    subject_tokens = _tokenize(subject_name)
    teacher_tokens = _tokenize(teacher_name)
    description_tokens = _tokenize(description)
    all_tokens = title_tokens + subject_tokens + teacher_tokens + description_tokens

    if title.startswith(query):
        return 6.0, "title_prefix"
    if any(token.startswith(query) for token in title_tokens):
        return 5.4, "title_token_prefix"
    if query in title:
        return 4.6, "title_contains"

    if subject_name.startswith(query):
        return 4.0, "subject_prefix"
    if any(token.startswith(query) for token in subject_tokens):
        return 3.6, "subject_token_prefix"
    if query in subject_name:
        return 3.1, "subject_contains"

    if any(token.startswith(query) for token in teacher_tokens):
        return 2.8, "teacher_prefix"
    if query in teacher_name:
        return 2.2, "teacher_contains"

    best_token_similarity = max((_fuzzy_similarity(query, token) for token in all_tokens), default=0.0)
    if best_token_similarity >= 0.92:
        return 2.6 + best_token_similarity, "fuzzy_token"
    if best_token_similarity >= 0.82:
        return 1.8 + best_token_similarity, "near_token"

    if query in description:
        return 1.2, "description_contains"

    return 0.0, "none"


def _autocomplete_subject_score(subject_name: str, search: str) -> float:
    normalized_subject = _normalize_text(subject_name)
    query = _normalize_text(search)
    if not normalized_subject or not query:
        return 0.0
    if normalized_subject.startswith(query):
        return 4.0
    if any(token.startswith(query) for token in _tokenize(normalized_subject)):
        return 3.4
    if query in normalized_subject:
        return 2.8
    best_similarity = max((_fuzzy_similarity(query, token) for token in _tokenize(normalized_subject)), default=0.0)
    if best_similarity >= 0.84:
        return 2.0 + best_similarity
    return 0.0


def _rank_course(
    course: Dict[str, Any],
    similarity_score: float,
    user_subjects: set[str],
    cluster_affinity: Dict[str, float],
) -> Tuple[float, float, float, float]:
    subject_name = _normalize_text(course.get("subjectName"))
    subject_boost = 0.01 if subject_name and subject_name in user_subjects else 0.0
    cluster_boost = min(cluster_affinity.get(str(course.get("id")), 0.0), 1.0) * 0.008
    popularity_score = min(_safe_float(course.get("enrolledStudents")) / 1000.0, 1.0)
    teacher_quality = min(_safe_float(course.get("teacherRating")) / 5.0, 1.0)

    search_score = similarity_score + subject_boost + cluster_boost + (0.005 * popularity_score) + (0.002 * teacher_quality)
    return search_score, subject_boost, cluster_boost, popularity_score


def _project_course_result(
    course: Dict[str, Any],
    search_score: float,
    similarity_score: float,
    match_source: str,
) -> Dict[str, Any]:
    item = dict(course)
    item["searchScore"] = round(search_score, 4)
    item["similarityScore"] = round(similarity_score, 4)
    item["matchSource"] = match_source
    return item


async def _read_cache(cache_key: str) -> Optional[Dict[str, Any]]:
    try:
        cached = await redis_conn.get(cache_key)
        if cached:
            return json.loads(cached)
    except Exception as exc:
        logger.warning(f"Course search cache read failed: {exc}")
    return None


async def _write_cache(cache_key: str, data: Dict[str, Any]) -> None:
    try:
        await redis_conn.setex(cache_key, settings.RETRIEVAL_CACHE_TTL, json.dumps(data))
    except Exception as exc:
        logger.warning(f"Course search cache write failed: {exc}")


async def semantic_course_search(
    user_id: str,
    search: str,
    filters: Dict[str, Any],
    page: int = _DEFAULT_PAGE,
    limit: int = _DEFAULT_LIMIT,
) -> Dict[str, Any]:
    page = max(page, 1)
    limit = min(max(limit, 1), _MAX_LIMIT)
    search = (search or "").strip()
    normalized_filters = dict(filters)
    cache_key = _search_cache_key(user_id, search, normalized_filters, page, limit)

    cached = await _read_cache(cache_key)
    if cached:
        return cached

    courses = await course_client.get_all_courses()
    catalog = _course_catalog_index(courses)
    user_profile = await course_client.get_user_analytics_profile(user_id)
    watched_subjects = _extract_subject_preferences(user_profile)
    cluster_affinity = _get_cluster_affinity(user_id)

    candidate_limit = min(max(limit * 5, 40), 200)
    hydrated_semantic: List[Dict[str, Any]] = []

    try:
        query_vector = await embedding_service.embed_text(search, normalize=True)
        if query_vector:
            await vector_store.ensure_collections(len(query_vector))
            semantic_hits = await vector_store.search_courses(query_vector, top_k=candidate_limit)
            for hit in semantic_hits:
                similarity = _safe_float(hit.get("similarityScore"))
                if similarity < _SEMANTIC_MIN_SCORE:
                    continue
                course = catalog.get(str(hit.get("courseId")))
                if not course or not _matches_filters(course, normalized_filters):
                    continue
                search_score, _, _, _ = _rank_course(course, similarity, watched_subjects, cluster_affinity)
                hydrated_semantic.append(
                    _project_course_result(course, search_score, similarity, "semantic")
                )
    except Exception as exc:
        logger.warning(f"Semantic course search failed for '{search}': {exc}")

    if hydrated_semantic:
        deduped: Dict[str, Dict[str, Any]] = {}
        for item in hydrated_semantic:
            cid = str(item.get("id"))
            if cid not in deduped or item["searchScore"] > deduped[cid]["searchScore"]:
                deduped[cid] = item
        ranked = sorted(
            deduped.values(),
            key=lambda item: (
                item.get("searchScore", 0.0),
                item.get("similarityScore", 0.0),
                _safe_int(item.get("enrolledStudents")),
                _safe_float(item.get("teacherRating")),
            ),
            reverse=True,
        )
    else:
        ranked = []

    if not ranked:
        fallback: List[Dict[str, Any]] = []
        for course in courses:
            if not _matches_filters(course, normalized_filters):
                continue
            keyword_score = _keyword_match_score(course, search)
            if keyword_score <= 0:
                continue
            fallback.append(
                _project_course_result(course, keyword_score, 0.0, "keyword_fallback")
            )
        ranked = sorted(
            fallback,
            key=lambda item: (
                item.get("searchScore", 0.0),
                _safe_int(item.get("enrolledStudents")),
                _safe_float(item.get("teacherRating")),
            ),
            reverse=True,
        )

    total = len(ranked)
    start = (page - 1) * limit
    paged = ranked[start:start + limit]
    response = {
        "data": paged,
        "meta": {
            "total": total,
            "page": page,
            "limit": limit,
        },
    }
    await _write_cache(cache_key, response)
    return response


async def course_autocomplete(user_id: str, search: str, limit: int = _DEFAULT_AUTOCOMPLETE_LIMIT) -> List[Dict[str, Any]]:
    search = (search or "").strip()
    if len(search) < _MIN_AUTOCOMPLETE_SEARCH_LENGTH:
        return []

    limit = min(max(limit, 1), _MAX_AUTOCOMPLETE_LIMIT)
    cache_key = _autocomplete_cache_key(user_id, search, limit)
    cached = await _read_cache(cache_key)
    if cached:
        return cached.get("data", [])

    courses = await course_client.get_all_courses()
    catalog = _course_catalog_index(courses)

    suggestions_by_course: Dict[str, Dict[str, Any]] = {}
    subject_scores: Dict[str, float] = {}
    lexical_floor = 0.1 if len(_normalize_text(search)) <= _SHORT_QUERY_LEN else 0.0

    try:
        query_vector = await embedding_service.embed_text(search, normalize=True)
        if query_vector:
            await vector_store.ensure_collections(len(query_vector))
            hits = await vector_store.search_courses(query_vector, top_k=min(max(limit * 4, 16), 80))
            for hit in hits:
                similarity = _safe_float(hit.get("similarityScore"))
                if similarity < _SEMANTIC_MIN_SCORE:
                    continue
                course = catalog.get(str(hit.get("courseId")))
                if not course:
                    continue
                lexical_score, lexical_source = _autocomplete_lexical_score(course, search)
                if lexical_score <= lexical_floor and len(_normalize_text(search)) <= _SHORT_QUERY_LEN:
                    continue
                final_score = lexical_score + (0.35 * similarity)
                course_id = str(course.get("id"))
                existing = suggestions_by_course.get(course_id)
                candidate = {
                    "type": "course",
                    "courseId": course_id,
                    "title": course.get("title"),
                    "subjectName": course.get("subjectName"),
                    "courseImage": course.get("courseImage"),
                    "score": round(final_score, 4),
                    "_lexicalScore": lexical_score,
                    "_semanticScore": similarity,
                    "_matchSource": lexical_source if lexical_score > 0 else "semantic",
                }
                if not existing or candidate["score"] > existing["score"]:
                    suggestions_by_course[course_id] = candidate
                subject_name = str(course.get("subjectName") or "").strip()
                if subject_name:
                    subject_score = _autocomplete_subject_score(subject_name, search)
                    semantic_subject_score = 0.15 * similarity
                    subject_scores[subject_name] = max(
                        subject_scores.get(subject_name, 0.0),
                        max(subject_score + semantic_subject_score, semantic_subject_score),
                    )
    except Exception as exc:
        logger.warning(f"Autocomplete semantic lookup failed for '{search}': {exc}")

    if not suggestions_by_course:
        for course in courses:
            lexical_score, lexical_source = _autocomplete_lexical_score(course, search)
            if lexical_score <= 0:
                continue
            suggestions_by_course[str(course.get("id"))] = {
                "type": "course",
                "courseId": str(course.get("id")),
                "title": course.get("title"),
                "subjectName": course.get("subjectName"),
                "courseImage": course.get("courseImage"),
                "score": round(lexical_score, 4),
                "_lexicalScore": lexical_score,
                "_semanticScore": 0.0,
                "_matchSource": lexical_source,
            }
            subject_name = str(course.get("subjectName") or "").strip()
            if subject_name:
                subject_score = _autocomplete_subject_score(subject_name, search)
                if subject_score > 0:
                    subject_scores[subject_name] = max(subject_scores.get(subject_name, 0.0), subject_score)

    course_suggestions = sorted(
        (
            {
                "type": item["type"],
                "courseId": item["courseId"],
                "title": item["title"],
                "subjectName": item["subjectName"],
                "courseImage": item["courseImage"],
                "score": item["score"],
            }
            for item in suggestions_by_course.values()
        ),
        key=lambda item: (item.get("score", 0.0), item.get("title") or ""),
        reverse=True,
    )

    subject_suggestions = sorted(
        (
            {"type": "subject", "subjectName": subject_name, "score": round(score, 4)}
            for subject_name, score in subject_scores.items()
        ),
        key=lambda item: (item.get("score", 0.0), item.get("subjectName") or ""),
        reverse=True,
    )

    combined = (course_suggestions[:limit] + subject_suggestions[:limit])[:limit]
    payload = {"data": combined}
    await _write_cache(cache_key, payload)
    return combined
