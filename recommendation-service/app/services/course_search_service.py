import hashlib
import json
import logging
import os
from datetime import datetime
from difflib import SequenceMatcher
from functools import lru_cache
from typing import Any, Dict, List, Optional, Set, Tuple

from app.clustering.cluster_service import ClusterService
from app.config import settings
from app.models.database import SessionLocal
from app.models.search import SearchFeedbackAggregate, SearchFeedbackEvent, SearchQueryAnalytics
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
_SEMANTIC_ONLY_MIN_CONFIDENCE = 0.72
_SEMANTIC_ONLY_TOP_GAP_MIN = 0.06
_SHORT_QUERY_LEN = 4
_MIN_CATALOG_TOKEN_LEN = 3
_MAX_CATALOG_CORRECTION_DISTANCE = 2
_CATALOG_TOKEN_SIMILARITY_MIN = 0.84
_CATALOG_PREFIX_SIMILARITY_MIN = 0.74
_CATALOG_PREFIX_EDIT_DISTANCE_MAX = 2
_RECENT_SEARCH_TTL = 7 * 24 * 60 * 60
_RECENT_SEARCH_LIMIT = 12
_SEARCH_FEEDBACK_TTL = 30 * 24 * 60 * 60

_QUERY_EXPANSIONS: Dict[str, List[str]] = {
    "hands-on": ["practical", "workshop", "lab", "bootcamp", "project based", "applied"],
    "hands on": ["practical", "workshop", "lab", "bootcamp", "project based", "applied"],
    "beginner": ["intro", "introduction", "fundamentals", "essentials", "starter"],
    "advanced": ["professional", "masterclass", "deep dive", "expert"],
    "real world": ["practical", "project based", "applied", "capstone"],
    "project based": ["project", "capstone", "lab", "workshop", "practical"],
    "interview prep": ["interview", "practice", "questions", "assessment"],
    "fast": ["accelerated", "quick start", "essentials", "crash course"],
}

_SEARCH_FEEDBACK_WEIGHTS: Dict[str, float] = {
    "click": 1.0,
    "preview": 1.25,
    "watch": 1.5,
    "enroll": 2.5,
}


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
    return f"course-search:v2:{hashlib.sha256(raw).hexdigest()}"


def _autocomplete_cache_key(user_id: str, search: str, limit: int) -> str:
    raw = json.dumps(
        {"user_id": user_id, "search": search.strip(), "limit": limit},
        sort_keys=True,
    ).encode("utf-8")
    return f"course-autocomplete:v2:{hashlib.sha256(raw).hexdigest()}"


def _recent_searches_key(user_id: str) -> str:
    return f"course-search:recent:{user_id}"


def _search_feedback_key(query: str) -> str:
    normalized = _normalize_text(query)
    digest = hashlib.sha256(normalized.encode("utf-8")).hexdigest()
    return f"course-search:feedback:{digest}"


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


def _dedupe_preserve_order(items: List[str]) -> List[str]:
    ordered: List[str] = []
    seen = set()
    for item in items:
        clean = _normalize_text(item)
        if not clean or clean in seen:
            continue
        seen.add(clean)
        ordered.append(clean)
    return ordered


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


def _query_variants(search: str) -> List[str]:
    normalized = _normalize_text(search)
    if not normalized:
        return []

    variants: List[str] = [normalized]

    for key, expansions in _QUERY_EXPANSIONS.items():
        if key in normalized:
            variants.extend(expansions)

    tokens = _tokenize(normalized)
    for token in tokens:
        variants.extend(_QUERY_EXPANSIONS.get(token, []))

    return _dedupe_preserve_order(variants)


@lru_cache(maxsize=8192)
def _levenshtein_distance(left: str, right: str) -> int:
    if left == right:
        return 0
    if not left:
        return len(right)
    if not right:
        return len(left)

    previous = list(range(len(right) + 1))
    for i, left_char in enumerate(left, start=1):
        current = [i]
        for j, right_char in enumerate(right, start=1):
            cost = 0 if left_char == right_char else 1
            current.append(
                min(
                    previous[j] + 1,
                    current[j - 1] + 1,
                    previous[j - 1] + cost,
                )
            )
        previous = current
    return previous[-1]


@lru_cache(maxsize=8192)
def _damerau_levenshtein_distance(left: str, right: str) -> int:
    if left == right:
        return 0
    if not left:
        return len(right)
    if not right:
        return len(left)

    rows = len(left) + 1
    cols = len(right) + 1
    dp = [[0] * cols for _ in range(rows)]

    for i in range(rows):
        dp[i][0] = i
    for j in range(cols):
        dp[0][j] = j

    for i in range(1, rows):
        for j in range(1, cols):
            cost = 0 if left[i - 1] == right[j - 1] else 1
            dp[i][j] = min(
                dp[i - 1][j] + 1,
                dp[i][j - 1] + 1,
                dp[i - 1][j - 1] + cost,
            )

            if (
                i > 1
                and j > 1
                and left[i - 1] == right[j - 2]
                and left[i - 2] == right[j - 1]
            ):
                dp[i][j] = min(dp[i][j], dp[i - 2][j - 2] + 1)

    return dp[-1][-1]


def _collect_catalog_terms(courses: List[Dict[str, Any]]) -> Set[str]:
    terms: Set[str] = set()
    for course in courses:
        for field in (
            course.get("title"),
            course.get("subjectName"),
            course.get("description"),
            course.get("teacherName"),
        ):
            for token in _tokenize(field):
                if len(token) >= _MIN_CATALOG_TOKEN_LEN:
                    terms.add(token)
    return terms


def _catalog_phrase_aliases(courses: List[Dict[str, Any]]) -> Dict[str, str]:
    aliases: Dict[str, str] = {}
    phrase_fields = ("subjectName", "title")

    for course in courses:
        for field_name in phrase_fields:
            phrase = _normalize_text(course.get(field_name))
            tokens = [token for token in _tokenize(phrase) if len(token) >= 2]
            if len(tokens) < 2:
                continue

            initials = "".join(token[0] for token in tokens)
            if len(initials) >= 2:
                aliases.setdefault(initials, phrase)

            compact = "".join(tokens)
            if len(compact) >= 6:
                aliases.setdefault(compact, phrase)

            for size in range(2, min(4, len(tokens)) + 1):
                partial_tokens = tokens[:size]
                partial_phrase = " ".join(partial_tokens)
                partial_initials = "".join(token[0] for token in partial_tokens)
                if len(partial_initials) >= 2:
                    aliases.setdefault(partial_initials, partial_phrase)

    return aliases


def _catalog_token_aliases(catalog_terms: Set[str]) -> Dict[str, str]:
    aliases: Dict[str, str] = {}
    for term in sorted(catalog_terms):
        if len(term) < _MIN_CATALOG_TOKEN_LEN:
            continue

        prefix_len = min(max(_MIN_CATALOG_TOKEN_LEN, len(term) - 2), len(term) - 1)
        if prefix_len >= _MIN_CATALOG_TOKEN_LEN:
            aliases.setdefault(term[:prefix_len], term)

        if len(term) >= 5:
            aliases.setdefault(term[:-1], term)

    return aliases


def _catalog_aliases(courses: List[Dict[str, Any]], catalog_terms: Set[str]) -> Dict[str, str]:
    aliases = _catalog_phrase_aliases(courses)
    aliases.update({key: value for key, value in _catalog_token_aliases(catalog_terms).items() if key not in aliases})
    return aliases


def _prefix_windows(candidate: str, token_len: int) -> List[str]:
    if not candidate or token_len <= 0:
        return []
    windows: List[str] = []
    min_len = max(_MIN_CATALOG_TOKEN_LEN, token_len - 1)
    max_len = min(len(candidate), token_len + 2)
    for size in range(min_len, max_len + 1):
        windows.append(candidate[:size])
    return _dedupe_preserve_order(windows)


def _best_catalog_term(token: str, catalog_terms: Optional[Set[str]]) -> str:
    normalized_token = _normalize_text(token)
    if not normalized_token or not catalog_terms or len(normalized_token) < _MIN_CATALOG_TOKEN_LEN:
        return normalized_token
    if normalized_token in catalog_terms:
        return normalized_token

    best_term = normalized_token
    best_similarity = 0.0
    best_distance: Optional[int] = None

    for candidate in catalog_terms:
        distance = _levenshtein_distance(normalized_token, candidate)
        damerau_distance = _damerau_levenshtein_distance(normalized_token, candidate)

        similarity = _fuzzy_similarity(normalized_token, candidate)
        if candidate.startswith(normalized_token) or normalized_token.startswith(candidate):
            similarity = max(similarity, 0.96)

        common_prefix_len = len(os.path.commonprefix([normalized_token, candidate]))
        prefix_similarity = 0.0
        prefix_distance: Optional[int] = None
        if len(normalized_token) >= 4:
            for prefix_window in _prefix_windows(candidate, len(normalized_token)):
                window_similarity = _fuzzy_similarity(normalized_token, prefix_window)
                window_distance = _levenshtein_distance(normalized_token, prefix_window)
                window_damerau_distance = _damerau_levenshtein_distance(normalized_token, prefix_window)
                if (
                    window_similarity > prefix_similarity
                    or (
                        window_similarity == prefix_similarity
                        and (
                            prefix_distance is None
                            or min(window_distance, window_damerau_distance) < prefix_distance
                        )
                    )
                ):
                    prefix_similarity = window_similarity
                    prefix_distance = min(window_distance, window_damerau_distance)

        allow_standard_match = min(distance, damerau_distance) <= _MAX_CATALOG_CORRECTION_DISTANCE and similarity >= _CATALOG_TOKEN_SIMILARITY_MIN
        allow_close_edit_match = (
            len(normalized_token) >= 5
            and len(candidate) >= 5
            and min(distance, damerau_distance) <= _MAX_CATALOG_CORRECTION_DISTANCE
            and normalized_token[0] == candidate[0]
            and similarity >= 0.66
        )
        allow_prefix_match = (
            common_prefix_len >= 2
            and prefix_similarity >= _CATALOG_PREFIX_SIMILARITY_MIN
            and (prefix_distance is not None and prefix_distance <= _CATALOG_PREFIX_EDIT_DISTANCE_MAX)
        )

        if not allow_standard_match and not allow_close_edit_match and not allow_prefix_match:
            continue

        if (
            best_distance is None
            or min(distance, damerau_distance) < best_distance
            or (
                min(distance, damerau_distance) == best_distance
                and max(similarity, prefix_similarity) > best_similarity
            )
            or (
                min(distance, damerau_distance) == best_distance
                and max(similarity, prefix_similarity) == best_similarity
                and abs(len(candidate) - len(normalized_token)) < abs(len(best_term) - len(normalized_token))
            )
        ):
            best_term = candidate
            best_similarity = max(similarity, prefix_similarity)
            best_distance = min(distance, damerau_distance)

    return best_term


def _canonicalize_query_token_with_catalog(
    token: str,
    catalog_terms: Optional[Set[str]] = None,
    catalog_aliases: Optional[Dict[str, str]] = None,
) -> str:
    normalized_token = _normalize_text(token)
    if not normalized_token:
        return normalized_token

    if catalog_aliases:
        alias = catalog_aliases.get(normalized_token)
        if alias:
            return alias

    if " " in normalized_token:
        return normalized_token
    return _best_catalog_term(normalized_token, catalog_terms)


def _normalized_query_forms(
    search: str,
    catalog_terms: Optional[Set[str]] = None,
    catalog_aliases: Optional[Dict[str, str]] = None,
) -> Dict[str, Any]:
    normalized = _normalize_text(search)
    raw_tokens = _tokenize(normalized)
    canonical_tokens = [
        _canonicalize_query_token_with_catalog(token, catalog_terms, catalog_aliases)
        for token in raw_tokens
    ]
    corrected = " ".join(canonical_tokens) if canonical_tokens else normalized
    base_text = corrected or normalized
    variants = _query_variants(base_text)
    variants = _dedupe_preserve_order(([normalized] if normalized else []) + ([corrected] if corrected else []) + variants)
    return {
        "normalized": normalized,
        "corrected": corrected or normalized,
        "tokens": raw_tokens,
        "canonical_tokens": canonical_tokens,
        "variants": variants,
    }


def _semantic_query_text(
    search: str,
    catalog_terms: Optional[Set[str]] = None,
    catalog_aliases: Optional[Dict[str, str]] = None,
) -> str:
    forms = _normalized_query_forms(search, catalog_terms, catalog_aliases)
    variants = forms["variants"]
    return ", ".join(variants) if variants else forms["normalized"]


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


def _best_lexical_score(
    course: Dict[str, Any],
    search: str,
    catalog_terms: Optional[Set[str]] = None,
    catalog_aliases: Optional[Dict[str, str]] = None,
) -> Tuple[float, str]:
    forms = _normalized_query_forms(search, catalog_terms, catalog_aliases)
    variants = forms["variants"] or [forms["normalized"]]
    best_score = 0.0
    best_source = "none"

    for index, variant in enumerate(variants):
        score, source = _autocomplete_lexical_score(course, variant)
        weight = 1.0 if index == 0 else 0.72
        adjusted = score * weight
        if adjusted > best_score:
            best_score = adjusted
            best_source = source if index == 0 else f"expanded:{source}"
    return best_score, best_source


def _best_keyword_score(
    course: Dict[str, Any],
    search: str,
    catalog_terms: Optional[Set[str]] = None,
    catalog_aliases: Optional[Dict[str, str]] = None,
) -> float:
    forms = _normalized_query_forms(search, catalog_terms, catalog_aliases)
    variants = forms["variants"] or [forms["normalized"]]
    best = 0.0
    for index, variant in enumerate(variants):
        score = _keyword_match_score(course, variant)
        if index > 0:
            score *= 0.72
        best = max(best, score)
    return best


def _best_query_token_match(query_token: str, searchable_tokens: List[str]) -> float:
    best = 0.0
    for token in searchable_tokens:
        if not token:
            continue
        if token.startswith(query_token):
            best = max(best, 1.0)
            continue
        if query_token in token:
            best = max(best, 0.95)
            continue
        best = max(best, _fuzzy_similarity(query_token, token))
    return best


def _token_coverage_metrics(
    course: Dict[str, Any],
    search: str,
    catalog_terms: Optional[Set[str]] = None,
    catalog_aliases: Optional[Dict[str, str]] = None,
) -> Tuple[int, int, float]:
    query_tokens = _normalized_query_forms(search, catalog_terms, catalog_aliases)["canonical_tokens"]
    if not query_tokens:
        return 0, 0, 0.0

    title = _normalize_text(course.get("title"))
    description = _normalize_text(course.get("description"))
    subject_name = _normalize_text(course.get("subjectName"))
    teacher_name = _normalize_text(course.get("teacherName"))
    searchable_tokens = _tokenize(f"{title} {subject_name} {teacher_name} {description}")

    matched_tokens = 0
    strong_matches = 0
    seen_query_tokens = set()

    for query_token in query_tokens:
        if query_token in seen_query_tokens:
            continue
        seen_query_tokens.add(query_token)

        best_similarity = _best_query_token_match(query_token, searchable_tokens)
        if best_similarity >= 0.82:
            matched_tokens += 1
        if best_similarity >= 0.92:
            strong_matches += 1

    coverage_ratio = matched_tokens / max(len(seen_query_tokens), 1)
    return matched_tokens, strong_matches, coverage_ratio


def _token_coverage_adjustment(
    course: Dict[str, Any],
    search: str,
    catalog_terms: Optional[Set[str]] = None,
    catalog_aliases: Optional[Dict[str, str]] = None,
) -> float:
    query_tokens = _normalized_query_forms(search, catalog_terms, catalog_aliases)["canonical_tokens"]
    unique_query_tokens = list(dict.fromkeys(query_tokens))
    if len(unique_query_tokens) < 2:
        return 0.0

    matched_tokens, strong_matches, coverage_ratio = _token_coverage_metrics(course, search, catalog_terms, catalog_aliases)
    if matched_tokens == 0:
        return -0.08
    if coverage_ratio >= 1.0:
        return 0.28 + (0.05 * strong_matches)
    if coverage_ratio >= 0.5:
        return -0.08 if strong_matches == 0 else 0.03 * strong_matches
    return -0.06


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


async def log_recent_search(user_id: str, search: str) -> None:
    normalized = _normalize_text(search)
    if not normalized:
        return
    try:
        existing = await get_recent_searches(user_id)
        merged = [normalized] + [item for item in existing if item != normalized]
        await redis_conn.setex(
            _recent_searches_key(user_id),
            _RECENT_SEARCH_TTL,
            json.dumps(merged[:_RECENT_SEARCH_LIMIT]),
        )
    except Exception as exc:
        logger.warning(f"Failed to log recent search for {user_id}: {exc}")


def _upsert_query_analytics(
    normalized_query: str,
    display_query: str,
    *,
    search_increment: int = 0,
    zero_result_increment: int = 0,
    event_type: Optional[str] = None,
) -> None:
    db = SessionLocal()
    if db is None:
        return
    try:
        analytics = (
            db.query(SearchQueryAnalytics)
            .filter(SearchQueryAnalytics.normalized_query == normalized_query)
            .first()
        )
        now = datetime.utcnow()
        if not analytics:
            analytics = SearchQueryAnalytics(
                normalized_query=normalized_query,
                display_query=display_query,
                total_searches=0,
                zero_result_searches=0,
                total_clicks=0,
                total_previews=0,
                total_watches=0,
                total_enrolls=0,
                last_searched_at=now,
                last_feedback_at=None,
            )
            db.add(analytics)

        if search_increment:
            analytics.total_searches += search_increment
            analytics.last_searched_at = now
            if display_query:
                analytics.display_query = display_query

        if zero_result_increment:
            analytics.zero_result_searches += zero_result_increment

        if event_type:
            if event_type == "click":
                analytics.total_clicks += 1
            elif event_type == "preview":
                analytics.total_previews += 1
            elif event_type == "watch":
                analytics.total_watches += 1
            elif event_type == "enroll":
                analytics.total_enrolls += 1
            analytics.last_feedback_at = now

        db.commit()
    except Exception as exc:
        db.rollback()
        logger.warning(f"Failed to update search query analytics for '{display_query}': {exc}")
    finally:
        if hasattr(db, "close"):
            db.close()


async def track_search_analytics(search: str, result_count: int) -> None:
    normalized_query = _normalize_text(search)
    if not normalized_query:
        return

    _upsert_query_analytics(
        normalized_query,
        search.strip(),
        search_increment=1,
        zero_result_increment=1 if result_count == 0 else 0,
    )


async def get_recent_searches(user_id: str) -> List[str]:
    try:
        cached = await redis_conn.get(_recent_searches_key(user_id))
        if not cached:
            return []
        parsed = json.loads(cached)
        if isinstance(parsed, list):
            return [str(item).strip().lower() for item in parsed if str(item).strip()]
    except Exception as exc:
        logger.warning(f"Failed to read recent searches for {user_id}: {exc}")
    return []


async def record_search_feedback(query: str, course_id: str, event_type: str) -> None:
    normalized_query = _normalize_text(query)
    normalized_event = _normalize_text(event_type)
    if not normalized_query or not course_id or normalized_event not in _SEARCH_FEEDBACK_WEIGHTS:
        return

    key = _search_feedback_key(normalized_query)
    try:
        cached = await redis_conn.get(key)
        payload = json.loads(cached) if cached else {}
        if not isinstance(payload, dict):
            payload = {}

        course_stats = payload.get(str(course_id), {})
        if not isinstance(course_stats, dict):
            course_stats = {}

        course_stats["score"] = float(course_stats.get("score", 0.0)) + _SEARCH_FEEDBACK_WEIGHTS[normalized_event]
        course_stats["events"] = int(course_stats.get("events", 0)) + 1
        course_stats["lastEvent"] = normalized_event
        payload[str(course_id)] = course_stats

        await redis_conn.setex(key, _SEARCH_FEEDBACK_TTL, json.dumps(payload))
    except Exception as exc:
        logger.warning(f"Failed to record search feedback for '{query}': {exc}")

    db = SessionLocal()
    if db is None:
        _upsert_query_analytics(normalized_query, query.strip(), event_type=normalized_event)
        return
    try:
        now = datetime.utcnow()
        weight = _SEARCH_FEEDBACK_WEIGHTS[normalized_event]

        db.add(
            SearchFeedbackEvent(
                user_id=None,
                query=query.strip(),
                normalized_query=normalized_query,
                course_id=str(course_id),
                event_type=normalized_event,
                weight=weight,
                created_at=now,
            )
        )

        aggregate = (
            db.query(SearchFeedbackAggregate)
            .filter(
                SearchFeedbackAggregate.normalized_query == normalized_query,
                SearchFeedbackAggregate.course_id == str(course_id),
            )
            .first()
        )
        if not aggregate:
            aggregate = SearchFeedbackAggregate(
                normalized_query=normalized_query,
                course_id=str(course_id),
                total_score=0.0,
                total_events=0,
                click_count=0,
                preview_count=0,
                watch_count=0,
                enroll_count=0,
                last_event_at=now,
            )
            db.add(aggregate)

        aggregate.total_score += weight
        aggregate.total_events += 1
        aggregate.last_event_at = now
        if normalized_event == "click":
            aggregate.click_count += 1
        elif normalized_event == "preview":
            aggregate.preview_count += 1
        elif normalized_event == "watch":
            aggregate.watch_count += 1
        elif normalized_event == "enroll":
            aggregate.enroll_count += 1

        db.commit()
    except Exception as exc:
        if hasattr(db, "rollback"):
            db.rollback()
        logger.warning(f"Failed to persist search feedback for '{query}': {exc}")
    finally:
        if hasattr(db, "close"):
            db.close()

    _upsert_query_analytics(normalized_query, query.strip(), event_type=normalized_event)


async def get_search_feedback_boosts(search: str) -> Dict[str, float]:
    normalized_query = _normalize_text(search)
    if not normalized_query:
        return {}

    key = _search_feedback_key(normalized_query)
    combined_scores: Dict[str, float] = {}
    try:
        cached = await redis_conn.get(key)
        if cached:
            payload = json.loads(cached)
            if isinstance(payload, dict):
                for course_id, stats in payload.items():
                    if not isinstance(stats, dict):
                        continue
                    combined_scores[str(course_id)] = max(
                        combined_scores.get(str(course_id), 0.0),
                        _safe_float(stats.get("score")),
                    )
    except Exception as exc:
        logger.warning(f"Failed to read search feedback boosts for '{search}': {exc}")

    db = SessionLocal()
    if db is None:
        max_score = max(combined_scores.values(), default=0.0) or 1.0
        return {
            course_id: min(score / max_score, 1.0)
            for course_id, score in combined_scores.items()
        }
    try:
        rows = (
            db.query(SearchFeedbackAggregate)
            .filter(SearchFeedbackAggregate.normalized_query == normalized_query)
            .all()
        )
        for row in rows:
            course_id = str(row.course_id)
            combined_scores[course_id] = max(combined_scores.get(course_id, 0.0), _safe_float(row.total_score))
    except Exception as exc:
        logger.warning(f"Failed to read persistent search feedback boosts for '{search}': {exc}")
    finally:
        if hasattr(db, "close"):
            db.close()

    max_score = max(combined_scores.values(), default=0.0) or 1.0
    boosts: Dict[str, float] = {}
    for course_id, score in combined_scores.items():
        boosts[course_id] = min(score / max_score, 1.0)
    return boosts


def _recent_search_boost(course: Dict[str, Any], recent_searches: List[str]) -> float:
    if not recent_searches:
        return 0.0
    best = 0.0
    for recent in recent_searches[:5]:
        lexical, _ = _best_lexical_score(course, recent)
        if lexical > 0:
            best = max(best, min(lexical / 12.0, 1.0))
    return 0.04 * best


def _query_feedback_boost(course: Dict[str, Any], feedback_boosts: Dict[str, float]) -> float:
    if not feedback_boosts:
        return 0.0
    score = feedback_boosts.get(str(course.get("id")), 0.0)
    return 0.18 * min(max(score, 0.0), 1.0)


def _is_low_confidence_semantic_only_query(
    ranked: List[Dict[str, Any]],
    search: str,
    catalog_terms: Optional[Set[str]] = None,
    catalog_aliases: Optional[Dict[str, str]] = None,
) -> bool:
    normalized_query = _normalize_text(search)
    if not normalized_query or not ranked:
        return False

    lexical_signals = []
    keyword_signals = []
    for item in ranked[:5]:
        lexical_score, _ = _best_lexical_score(item, normalized_query, catalog_terms, catalog_aliases)
        keyword_score = _best_keyword_score(item, normalized_query, catalog_terms, catalog_aliases)
        lexical_signals.append(lexical_score)
        keyword_signals.append(keyword_score)

    if max(lexical_signals, default=0.0) > 0 or max(keyword_signals, default=0.0) > 0:
        return False

    top_similarity = _safe_float(ranked[0].get("similarityScore"))
    second_similarity = _safe_float(ranked[1].get("similarityScore")) if len(ranked) > 1 else 0.0
    similarity_gap = top_similarity - second_similarity

    return top_similarity < _SEMANTIC_ONLY_MIN_CONFIDENCE or similarity_gap < _SEMANTIC_ONLY_TOP_GAP_MIN


def _semantic_match_source(lexical_score: float, search_boost: float, cluster_boost: float) -> str:
    sources = ["semantic"]
    if lexical_score > 0:
        sources.append("lexical")
    if search_boost > 0:
        sources.append("recent_search")
    if cluster_boost > 0:
        sources.append("cluster")
    return "+".join(sources)


def _select_diverse_results(items: List[Dict[str, Any]], limit: int) -> List[Dict[str, Any]]:
    if len(items) <= limit:
        return items

    selected: List[Dict[str, Any]] = []
    subject_counts: Dict[str, int] = {}

    for item in items:
        if len(selected) >= limit:
            break
        subject = _normalize_text(item.get("subjectName")) or "__unknown__"
        count = subject_counts.get(subject, 0)
        if count >= 2:
            continue
        selected.append(item)
        subject_counts[subject] = count + 1

    if len(selected) < limit:
        selected_ids = {str(item.get("id")) for item in selected}
        for item in items:
            if len(selected) >= limit:
                break
            if str(item.get("id")) in selected_ids:
                continue
            selected.append(item)

    return selected


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
    catalog_terms = _collect_catalog_terms(courses)
    catalog_aliases = _catalog_aliases(courses, catalog_terms)
    user_profile = await course_client.get_user_analytics_profile(user_id)
    watched_subjects = _extract_subject_preferences(user_profile)
    cluster_affinity = _get_cluster_affinity(user_id)
    recent_searches = await get_recent_searches(user_id)
    query_feedback_boosts = await get_search_feedback_boosts(search)

    candidate_limit = min(max(limit * 5, 40), 200)
    hydrated_candidates: Dict[str, Dict[str, Any]] = {}

    try:
        query_vector = await embedding_service.embed_text(_semantic_query_text(search, catalog_terms, catalog_aliases), normalize=True)
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
                lexical_score, _ = _best_lexical_score(course, search, catalog_terms, catalog_aliases)
                search_score, _, cluster_boost, _ = _rank_course(course, similarity, watched_subjects, cluster_affinity)
                recent_boost = _recent_search_boost(course, recent_searches)
                feedback_boost = _query_feedback_boost(course, query_feedback_boosts)
                token_coverage_boost = _token_coverage_adjustment(course, search, catalog_terms, catalog_aliases)
                search_score += min(lexical_score / 20.0, 0.08) + recent_boost + feedback_boost + token_coverage_boost
                result = _project_course_result(
                    course,
                    search_score,
                    similarity,
                    _semantic_match_source(lexical_score, recent_boost + feedback_boost, cluster_boost),
                )
                course_id = str(course.get("id"))
                existing = hydrated_candidates.get(course_id)
                if not existing or result["searchScore"] > existing["searchScore"]:
                    hydrated_candidates[course_id] = result
    except Exception as exc:
        logger.warning(f"Semantic course search failed for '{search}': {exc}")

    for course in courses:
        if not _matches_filters(course, normalized_filters):
            continue
        lexical_score, lexical_source = _best_lexical_score(course, search, catalog_terms, catalog_aliases)
        keyword_score = _best_keyword_score(course, search, catalog_terms, catalog_aliases)
        if lexical_score <= 0 and keyword_score <= 0:
            continue
        course_id = str(course.get("id"))
        existing = hydrated_candidates.get(course_id)
        similarity = _safe_float(existing.get("similarityScore") if existing else 0.0)
        recent_boost = _recent_search_boost(course, recent_searches)
        feedback_boost = _query_feedback_boost(course, query_feedback_boosts)
        token_coverage_boost = _token_coverage_adjustment(course, search, catalog_terms, catalog_aliases)
        lexical_total = max(lexical_score, keyword_score) + (0.25 * similarity) + recent_boost + feedback_boost + token_coverage_boost
        candidate = _project_course_result(course, lexical_total, similarity, f"lexical:{lexical_source}")
        if not existing or candidate["searchScore"] > existing["searchScore"]:
            hydrated_candidates[course_id] = candidate

    if hydrated_candidates:
        ranked = sorted(
            hydrated_candidates.values(),
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

    if ranked and _is_low_confidence_semantic_only_query(ranked, search, catalog_terms, catalog_aliases):
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
    paged = _select_diverse_results(ranked[start:], limit)
    response = {
        "data": paged,
        "meta": {
            "total": total,
            "page": page,
            "limit": limit,
        },
    }
    await track_search_analytics(search, total)
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
    catalog_terms = _collect_catalog_terms(courses)
    catalog_aliases = _catalog_aliases(courses, catalog_terms)
    normalized_forms = _normalized_query_forms(search, catalog_terms, catalog_aliases)
    subject_query = normalized_forms["corrected"]

    suggestions_by_course: Dict[str, Dict[str, Any]] = {}
    subject_scores: Dict[str, float] = {}
    lexical_floor = 0.1 if len(_normalize_text(search)) <= _SHORT_QUERY_LEN else 0.0
    query_feedback_boosts = await get_search_feedback_boosts(search)

    try:
        query_vector = await embedding_service.embed_text(_semantic_query_text(search, catalog_terms, catalog_aliases), normalize=True)
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
                lexical_score, lexical_source = _best_lexical_score(course, search, catalog_terms, catalog_aliases)
                if lexical_score <= lexical_floor and len(_normalize_text(search)) <= _SHORT_QUERY_LEN:
                    continue
                feedback_boost = _query_feedback_boost(course, query_feedback_boosts)
                final_score = lexical_score + (0.35 * similarity) + feedback_boost
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
                    subject_score = _autocomplete_subject_score(subject_name, subject_query)
                    semantic_subject_score = 0.15 * similarity
                    subject_scores[subject_name] = max(
                        subject_scores.get(subject_name, 0.0),
                        max(subject_score + semantic_subject_score, semantic_subject_score),
                    )
    except Exception as exc:
        logger.warning(f"Autocomplete semantic lookup failed for '{search}': {exc}")

    if not suggestions_by_course:
        for course in courses:
            lexical_score, lexical_source = _best_lexical_score(course, search, catalog_terms, catalog_aliases)
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
                subject_score = _autocomplete_subject_score(subject_name, subject_query)
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


def get_top_clicked_queries(limit: int = 10) -> List[Dict[str, Any]]:
    db = SessionLocal()
    if db is None:
        return []
    try:
        query = db.query(SearchQueryAnalytics)
        if hasattr(SearchQueryAnalytics.total_clicks, "desc"):
            query = query.order_by(
                SearchQueryAnalytics.total_clicks.desc(),
                SearchQueryAnalytics.total_previews.desc(),
                SearchQueryAnalytics.total_watches.desc(),
                SearchQueryAnalytics.last_feedback_at.desc().nullslast(),
            )
        rows = query.limit(limit).all()
        return [
            {
                "query": row.display_query,
                "totalSearches": row.total_searches,
                "zeroResultSearches": row.zero_result_searches,
                "zeroResultRate": round((row.zero_result_searches / row.total_searches) if row.total_searches else 0.0, 4),
                "totalClicks": row.total_clicks,
                "totalPreviews": row.total_previews,
                "totalWatches": row.total_watches,
                "totalEnrolls": row.total_enrolls,
                "lastSearchedAt": row.last_searched_at.isoformat() if row.last_searched_at else None,
                "lastFeedbackAt": row.last_feedback_at.isoformat() if row.last_feedback_at else None,
            }
            for row in rows
        ]
    finally:
        if hasattr(db, "close"):
            db.close()


def get_zero_result_queries(limit: int = 10) -> List[Dict[str, Any]]:
    db = SessionLocal()
    if db is None:
        return []
    try:
        query = db.query(SearchQueryAnalytics)
        if hasattr(SearchQueryAnalytics.zero_result_searches, "__gt__"):
            try:
                query = query.filter(SearchQueryAnalytics.zero_result_searches > 0)
            except TypeError:
                pass
        if hasattr(SearchQueryAnalytics.zero_result_searches, "desc"):
            query = query.order_by(
                SearchQueryAnalytics.zero_result_searches.desc(),
                SearchQueryAnalytics.last_searched_at.desc(),
            )
        rows = query.limit(limit).all()
        return [
            {
                "query": row.display_query,
                "totalSearches": row.total_searches,
                "zeroResultSearches": row.zero_result_searches,
                "zeroResultRate": round((row.zero_result_searches / row.total_searches) if row.total_searches else 0.0, 4),
                "totalClicks": row.total_clicks,
                "totalPreviews": row.total_previews,
                "totalWatches": row.total_watches,
                "totalEnrolls": row.total_enrolls,
                "lastSearchedAt": row.last_searched_at.isoformat() if row.last_searched_at else None,
                "lastFeedbackAt": row.last_feedback_at.isoformat() if row.last_feedback_at else None,
            }
            for row in rows
        ]
    finally:
        if hasattr(db, "close"):
            db.close()


def get_top_query_course_pairs(limit: int = 10) -> List[Dict[str, Any]]:
    db = SessionLocal()
    if db is None:
        return []
    try:
        query = db.query(SearchFeedbackAggregate)
        if hasattr(SearchFeedbackAggregate.total_score, "desc"):
            query = query.order_by(
                SearchFeedbackAggregate.total_score.desc(),
                SearchFeedbackAggregate.total_events.desc(),
                SearchFeedbackAggregate.last_event_at.desc(),
            )
        rows = query.limit(limit).all()
        return [
            {
                "query": row.normalized_query,
                "courseId": row.course_id,
                "totalScore": round(_safe_float(row.total_score), 4),
                "totalEvents": row.total_events,
                "clickCount": row.click_count,
                "previewCount": row.preview_count,
                "watchCount": row.watch_count,
                "enrollCount": row.enroll_count,
                "lastEventAt": row.last_event_at.isoformat() if row.last_event_at else None,
            }
            for row in rows
        ]
    finally:
        if hasattr(db, "close"):
            db.close()
