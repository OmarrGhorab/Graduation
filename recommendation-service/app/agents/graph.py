import logging
from typing import Any, Dict, List

from langgraph.graph import END, StateGraph
from opentelemetry import trace

from app.agents.prompts import RANKER_SYSTEM_PROMPT
from app.agents.state import RecommendationState
from app.config import settings
from app.services.gemma_client import gemma_client
from app.tools.registry import tool_registry
from app.retrieval.hybrid_search import search_relevant_courses

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
        query = user_history.get("behavior_query") or ", ".join(interests) or "popular courses"
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

        user_history = state["context"].get("get_user_history", {})

        # Pass candidates by index so the LLM never has to copy UUIDs
        indexed = [
            {
                "idx": i,
                "title": c.get("title"),
                "subject": c.get("subjectName"),
                "hybridScore": round(c.get("hybridScore", 0), 3),
            }
            for i, c in enumerate(candidates)
        ]
        payload = {
            "user_id": state["user_id"],
            "interests": user_history.get("interests") or [],
            "cart_subjects": user_history.get("cart_subjects") or [],
            "subject_preferences": user_history.get("subject_preferences") or [],
            "candidates": indexed,
            "top_n": settings.AGENT_FINAL_RECOMMENDATION_COUNT,
        }

        ranked = await gemma_client.rank_recommendation_candidates(
            system_prompt=RANKER_SYSTEM_PROMPT,
            payload=payload,
        )

        if not isinstance(ranked, list) or not ranked:
            state["errors"].append("LLM ranking returned invalid output")
            span.set_attribute("recommendation.fallback", True)
            return fallback_ranker(state)

        result = []
        for item in ranked:
            idx = item.get("idx")
            if idx is None or not isinstance(idx, int) or idx >= len(candidates):
                continue
            merged = {
                **candidates[idx],
                "matchReason": item.get("matchReason", ""),
                "priority": item.get("priority", "MEDIUM"),
            }
            merged["score"] = _display_score(merged)
            result.append(merged)
            if len(result) >= settings.AGENT_FINAL_RECOMMENDATION_COUNT:
                break

        state["recommendations"] = result
        span.set_attribute("recommendation.result_count", len(result))
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
        hybrid = item.get("hybridScore", 0)
        fallback_item = {
            **item,
            "matchReason": item.get("matchReason", "Matched by semantic similarity and popularity."),
            "priority": item.get("priority", "MEDIUM"),
            "source": item.get("source", ["semantic_similarity"]),
        }
        fallback_item["score"] = _display_score(fallback_item)
        fallback.append(fallback_item)
    state["recommendations"] = fallback
    return state


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
    query = user_history.get("behavior_query") or ", ".join(interests) or "recommended courses"
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
