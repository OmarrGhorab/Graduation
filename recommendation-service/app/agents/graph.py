import logging
from typing import Any, Dict, List

from langgraph.graph import END, StateGraph
from opentelemetry import trace

from app.agents.state import RecommendationState
from app.config import settings
from app.services.course_client import course_client
from app.tools.registry import tool_registry
from app.retrieval.hybrid_search import search_relevant_courses
from app.utils.profile_utils import get_subject_name

logger = logging.getLogger(__name__)
tracer = trace.get_tracer(__name__)


def _initial_state(user_id: str) -> RecommendationState:
    return RecommendationState(
        user_id=user_id,
        tool_calls=0,
        done=False,
        next_tool=None,
        next_tool_args={},
        tool_trace=[],
        context={},
        candidates=[],
        recommendations=[],
        errors=[],
        reasoning_summary="",
    )


async def plan_next_tool(state: RecommendationState) -> RecommendationState:
    if state["tool_calls"] >= settings.AGENT_MAX_TOOL_CALLS:
        state["done"] = True
        return state

    context = state["context"]
    has_user_history = "get_user_history" in context
    has_candidates = bool(state["candidates"])

    # Step 1: always fetch user history first
    if not has_user_history:
        state["done"] = False
        state["next_tool"] = "get_user_history"
        state["next_tool_args"] = {"user_id": state["user_id"]}
        state["reasoning_summary"] = "Fetching user enrollment history and interests."
        return state

    # Step 2: search with interests as query (only once)
    already_searched = any(t.get("tool") == "search_relevant_courses" for t in state["tool_trace"])
    if not has_candidates and not already_searched:
        user_history = context.get("get_user_history", {})
        enrolled_ids = user_history.get("enrolled_course_ids") or []
        interests = user_history.get("interests") or []
        query = ", ".join(interests) or user_history.get("behavior_query") or "popular courses"
        state["done"] = False
        state["next_tool"] = "search_relevant_courses"
        state["next_tool_args"] = {
            "user_id": state["user_id"],
            "query": query,
            "exclude_course_ids": enrolled_ids,
        }
        state["reasoning_summary"] = f"Searching with interests: {query[:80]}"
        return state

    # Step 3: enough context — let the ranker take over
    state["done"] = True
    state["next_tool"] = None
    state["next_tool_args"] = {}
    state["reasoning_summary"] = "Collected user history and search candidates. Ready to rank."
    return state


async def execute_tool(state: RecommendationState) -> RecommendationState:
    if state["done"] or not state["next_tool"]:
        return state

    response = await tool_registry.invoke(state["next_tool"], state["next_tool_args"])
    trace_item = {
        "tool": response.name,
        "success": response.success,
        "error": response.error,
        "data": response.data,
    }
    state["tool_trace"].append(trace_item)
    state["tool_calls"] += 1
    if not response.success and response.error:
        state["errors"].append(response.error)
    return state


async def merge_tool_result(state: RecommendationState) -> RecommendationState:
    if not state["tool_trace"]:
        return state

    last = state["tool_trace"][-1]
    if not last.get("success"):
        return state

    tool_name = last.get("tool")
    data = last.get("data") or {}

    if tool_name == "search_relevant_courses":
        courses = data.get("courses", [])
        state["candidates"] = _merge_candidates(state["candidates"], courses)
    elif tool_name == "get_trending_courses":
        state["context"]["trending"] = data.get("courses", [])
    else:
        state["context"][tool_name] = data

    return state


def should_continue(state: RecommendationState) -> str:
    if state["done"]:
        return "rank"
    if state["tool_calls"] >= settings.AGENT_MAX_TOOL_CALLS:
        return "rank"
    if not state["next_tool"]:
        return "rank"
    return "execute"


def _merge_candidates(existing: List[Dict[str, Any]], incoming: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
    by_id = {str(item.get("courseId")): item for item in existing if item.get("courseId")}
    for item in incoming:
        cid = str(item.get("courseId"))
        if not cid:
            continue
        if cid in by_id:
            if item.get("hybridScore", 0) > by_id[cid].get("hybridScore", 0):
                by_id[cid] = item
        else:
            by_id[cid] = item
    return list(by_id.values())


async def rank_candidates(state: RecommendationState) -> RecommendationState:
    with tracer.start_as_current_span("recommendation.llm.rank") as span:
        candidates = sorted(state["candidates"], key=lambda x: x.get("hybridScore", 0), reverse=True)
        candidates = candidates[: settings.AGENT_TOP_K_CANDIDATES]
        span.set_attribute("recommendation.candidate_count", len(candidates))
        if not candidates:
            span.set_attribute("recommendation.fallback", True)
            return await fallback_retrieval(state)
        state = fallback_ranker(state)
        span.set_attribute("recommendation.result_count", len(state["recommendations"]))
        if not state["recommendations"]:
            span.set_attribute("recommendation.fallback", True)
            return await fallback_retrieval(state)
        span.set_attribute("recommendation.fallback", False)
        return state


def validate_output(state: RecommendationState) -> RecommendationState:
    valid = []
    for item in state["recommendations"]:
        if not item.get("courseId"):
            continue
        item["score"] = _display_score(item)
        if "matchReason" not in item:
            item["matchReason"] = "Recommended based on retrieval and ranking signals."
        if "priority" not in item:
            item["priority"] = "MEDIUM"
        if "source" not in item:
            item["source"] = ["semantic_similarity"]
        valid.append(item)
    state["recommendations"] = valid[: settings.AGENT_FINAL_RECOMMENDATION_COUNT]
    return state


def _display_score(item: Dict[str, Any]) -> int:
    hybrid = float(item.get("hybridScore", 0) or 0)
    priority = str(item.get("priority", "MEDIUM")).upper()
    priority_adjustment = {
        "HIGH": 0.10,
        "MEDIUM": 0.0,
        "LOW": -0.25,
    }.get(priority, 0.0)
    adjusted = max(0.0, min(1.0, hybrid + priority_adjustment))
    return int(adjusted * 100)


def fallback_ranker(state: RecommendationState) -> RecommendationState:
    ranked = sorted(state["candidates"], key=lambda x: x.get("hybridScore", 0), reverse=True)
    fallback = []
    for item in ranked[: settings.AGENT_FINAL_RECOMMENDATION_COUNT]:
        reason = _build_deterministic_reason(item)
        fallback_item = {
            **item,
            "matchReason": item.get("matchReason", reason),
            "priority": item.get("priority", _priority_from_score(float(item.get("hybridScore", 0) or 0))),
            "source": item.get("source", ["semantic_similarity"]),
        }
        fallback_item["score"] = _display_score(fallback_item)
        fallback.append(fallback_item)
    state["recommendations"] = fallback
    return state


def _priority_from_score(hybrid_score: float) -> str:
    if hybrid_score >= 0.78:
        return "HIGH"
    if hybrid_score >= 0.52:
        return "MEDIUM"
    return "LOW"


def _build_deterministic_reason(item: Dict[str, Any]) -> str:
    reasons: List[str] = []
    subject = item.get("subjectName")
    if float(item.get("interestBoost", 0) or 0) > 0:
        reasons.append("Matches your selected onboarding interests.")
    if subject and float(item.get("profileSubjectBoost", 0) or 0) > 0:
        reasons.append(f"Strong match with your watched interest in {subject}.")
    if float(item.get("clusterContribution", 0) or 0) > 0:
        reasons.append("Popular among learners with similar watch patterns.")
    if float(item.get("similarityScore", 0) or 0) >= 0.75:
        reasons.append("Close semantic match to your recent learning behavior.")
    if not reasons:
        reasons.append("Ranks well based on relevance, popularity, and instructor quality.")
    return " ".join(reasons)


def _normalize_text(value: Any) -> str:
    return str(value or "").strip().lower()


def _history_interest_terms(user_history: Dict[str, Any]) -> List[str]:
    terms: List[str] = []

    for value in user_history.get("interests", []) or []:
        if value:
            terms.append(str(value))

    for value in user_history.get("cart_subjects", []) or []:
        if value:
            terms.append(str(value))

    for item in user_history.get("preview_interests", []) or []:
        subject_name = get_subject_name(item)
        if subject_name:
            terms.append(subject_name)

    for item in user_history.get("subject_preferences", []) or []:
        subject_name = get_subject_name(item)
        if subject_name:
            terms.append(subject_name)

    seen = set()
    unique_terms = []
    for term in terms:
        normalized = _normalize_text(term)
        if not normalized or normalized in seen:
            continue
        seen.add(normalized)
        unique_terms.append(term)
    return unique_terms


def _course_interest_match_score(course: Dict[str, Any], terms: List[str]) -> float:
    if not terms:
        return 0.0

    haystack = " ".join(
        [
            _normalize_text(course.get("title")),
            _normalize_text(course.get("description")),
            _normalize_text(course.get("subjectName")),
            _normalize_text(course.get("teacherName")),
        ]
    )
    if not haystack:
        return 0.0

    score = 0.0
    for term in terms:
        normalized_term = _normalize_text(term)
        if not normalized_term:
            continue
        if normalized_term in haystack:
            score += 0.55
            continue
        tokens = [token for token in normalized_term.split() if len(token) >= 3]
        if tokens and all(token in haystack for token in tokens):
            score += 0.35
        elif any(token in haystack for token in tokens):
            score += 0.15
    return score


async def _profile_catalog_fallback(user_history: Dict[str, Any], exclude_set: set[str]) -> List[Dict[str, Any]]:
    terms = _history_interest_terms(user_history)
    if not terms:
        return []

    courses = await course_client.get_all_courses()
    ranked: List[Dict[str, Any]] = []
    for course in courses or []:
        course_id = str(course.get("id") or course.get("courseId") or "")
        if not course_id or course_id in exclude_set:
            continue
        interest_score = _course_interest_match_score(course, terms)
        if interest_score <= 0:
            continue

        popularity = min(float(course.get("enrollmentCount", 0) or 0) / 1000.0, 1.0)
        teacher_score = min(float(course.get("teacherAuthority", course.get("teacherScore", 0)) or 0) / 5.0, 1.0)
        hybrid_score = min(1.0, 0.72 * min(interest_score, 1.0) + 0.18 * popularity + 0.10 * teacher_score)
        subject_name = course.get("subjectName")
        ranked.append(
            {
                "courseId": course_id,
                "title": course.get("title"),
                "courseImage": course.get("courseImage"),
                "price": course.get("price", 0),
                "currency": course.get("currency", "EGP"),
                "enrolledCount": course.get("enrollmentCount", 0),
                "subjectName": subject_name,
                "teacher": {
                    "name": course.get("teacherName", "Recommendation Teacher"),
                    "avatar": course.get("teacherProfileImg") or "https://i.pravatar.cc/150?u=fallback",
                },
                "similarityScore": round(min(interest_score, 1.0), 4),
                "hybridScore": hybrid_score,
                "profileSubjectBoost": 0.0,
                "interestBoost": round(min(interest_score, 1.0), 4),
                "clusterContribution": 0.0,
                "source": ["profile_interest"],
                "matchReason": f"Matches your selected or early activity interest in {subject_name or terms[0]}.",
                "reason": f"Matches your selected or early activity interest in {subject_name or terms[0]}.",
                "priority": "HIGH",
            }
        )

    ranked.sort(key=lambda item: item.get("hybridScore", 0), reverse=True)
    return ranked[: settings.AGENT_TOP_K_CANDIDATES]


async def fallback_retrieval(state: RecommendationState) -> RecommendationState:
    from app.services.recommendation_engine import get_trending_recommendations
    from app.retrieval.hybrid_search import get_enrolled_course_ids

    user_history = next(
        (
            item.get("data", {})
            for item in reversed(state["tool_trace"])
            if item.get("tool") == "get_user_history" and item.get("success")
        ),
        {},
    )
    exclude_course_ids = [str(cid) for cid in user_history.get("enrolled_course_ids", []) if cid]

    # Always fetch enrolled IDs directly if the tool trace didn't have them
    if not exclude_course_ids:
        try:
            exclude_course_ids = await get_enrolled_course_ids(state["user_id"])
        except Exception as exc:
            logger.warning(f"Could not fetch enrolled IDs for fallback filter: {exc}")

    interests = user_history.get("interests", []) or []
    query = ", ".join(interests) or user_history.get("behavior_query") or "recommended courses"
    exclude_set = set(exclude_course_ids)

    try:
        semantic = await search_relevant_courses(
            user_id=state["user_id"],
            query=query,
            top_k=settings.AGENT_TOP_K_CANDIDATES,
            exclude_course_ids=exclude_course_ids,
        )
    except Exception as exc:
        logger.warning(f"Agent fallback semantic retrieval failed: {exc}")
        semantic = []

    if not semantic:
        try:
            semantic = await _profile_catalog_fallback(user_history, exclude_set)
        except Exception as exc:
            logger.warning(f"Agent fallback profile catalog ranking failed: {exc}")
            semantic = []

    if not semantic:
        try:
            trending = await get_trending_recommendations()
            for item in trending:
                if "hybridScore" not in item:
                    item["hybridScore"] = min(item.get("score", 0) / 100.0, 1.0)
            semantic = [c for c in trending if str(c.get("courseId", "")) not in exclude_set]
        except Exception as exc:
            logger.warning(f"Agent fallback trending retrieval failed: {exc}")
            semantic = []

    state["candidates"] = _merge_candidates(state["candidates"], semantic)
    if state["candidates"]:
        state = fallback_ranker(state)
    else:
        state["recommendations"] = []
    if not state["recommendations"] and semantic:
        state["recommendations"] = semantic[: settings.AGENT_FINAL_RECOMMENDATION_COUNT]
    return state


def build_recommendation_graph():
    graph = StateGraph(RecommendationState)

    graph.add_node("plan", plan_next_tool)
    graph.add_node("execute", execute_tool)
    graph.add_node("merge", merge_tool_result)
    graph.add_node("rank", rank_candidates)
    graph.add_node("validate", validate_output)

    graph.set_entry_point("plan")
    graph.add_conditional_edges("plan", should_continue, {"execute": "execute", "rank": "rank"})
    graph.add_edge("execute", "merge")
    graph.add_edge("merge", "plan")
    graph.add_edge("rank", "validate")
    graph.add_edge("validate", END)

    return graph.compile()


async def run_recommendation_graph(user_id: str) -> RecommendationState:
    app = build_recommendation_graph()
    state = _initial_state(user_id)
    recursion_limit = settings.AGENT_MAX_TOOL_CALLS * 3 + 10
    final_state = await app.ainvoke(state, config={"recursion_limit": recursion_limit})
    return final_state
