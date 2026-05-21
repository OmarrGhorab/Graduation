import logging
from typing import Any, Dict, List

from langgraph.graph import END, StateGraph
from opentelemetry import trace

from app.agents.prompts import RANKER_SYSTEM_PROMPT, TOOL_PLANNER_SYSTEM_PROMPT
from app.agents.state import RecommendationState
from app.config import settings
from app.services.gemma_client import gemma_client
from app.tools.registry import tool_registry

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

    tool_specs = tool_registry.list_tools()
    planner_input = {
        "user_id": state["user_id"],
        "tool_calls": state["tool_calls"],
        "context_keys": list(state["context"].keys()),
        "available_tools": tool_specs,
        "last_tool_result": state["tool_trace"][-1] if state["tool_trace"] else None,
    }

    plan = await gemma_client.plan_next_tool(
        system_prompt=TOOL_PLANNER_SYSTEM_PROMPT,
        payload=planner_input,
    )

    state["done"] = bool(plan.get("done", False))
    state["next_tool"] = plan.get("tool_name")
    state["next_tool_args"] = plan.get("arguments") or {}
    state["reasoning_summary"] = plan.get("reasoning_summary", state["reasoning_summary"])
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
            state["recommendations"] = []
            span.set_attribute("recommendation.result_count", 0)
            return state

        payload = {
            "user_id": state["user_id"],
            "context": state["context"],
            "tool_trace": state["tool_trace"][-5:],
            "candidates": candidates,
            "top_n": settings.AGENT_FINAL_RECOMMENDATION_COUNT,
        }

        ranked = await gemma_client.rank_recommendation_candidates(
            system_prompt=RANKER_SYSTEM_PROMPT,
            payload=payload,
        )

        if not isinstance(ranked, list):
            state["errors"].append("LLM ranking returned non-list output")
            span.set_attribute("recommendation.fallback", True)
            return fallback_ranker(state)

        by_id = {str(c.get("courseId")): c for c in candidates if c.get("courseId")}
        result = []
        for item in ranked:
            course_id = str(item.get("courseId", ""))
            if course_id not in by_id:
                continue
            merged = {**by_id[course_id], **item}
            result.append(merged)
            if len(result) >= settings.AGENT_FINAL_RECOMMENDATION_COUNT:
                break

        state["recommendations"] = result
        span.set_attribute("recommendation.result_count", len(result))
        if not state["recommendations"]:
            span.set_attribute("recommendation.fallback", True)
            return fallback_ranker(state)
        span.set_attribute("recommendation.fallback", False)
        return state


def validate_output(state: RecommendationState) -> RecommendationState:
    valid = []
    for item in state["recommendations"]:
        if not item.get("courseId"):
            continue
        if "score" not in item:
            item["score"] = int(max(0, min(100, item.get("hybridScore", 0) * 100)))
        if "matchReason" not in item:
            item["matchReason"] = "Recommended based on retrieval and ranking signals."
        if "priority" not in item:
            item["priority"] = "MEDIUM"
        if "source" not in item:
            item["source"] = ["semantic_similarity"]
        valid.append(item)
    state["recommendations"] = valid[: settings.AGENT_FINAL_RECOMMENDATION_COUNT]
    return state


def fallback_ranker(state: RecommendationState) -> RecommendationState:
    ranked = sorted(state["candidates"], key=lambda x: x.get("hybridScore", 0), reverse=True)
    fallback = []
    for item in ranked[: settings.AGENT_FINAL_RECOMMENDATION_COUNT]:
        fallback.append(
            {
                **item,
                "score": int(max(0, min(100, item.get("hybridScore", 0) * 100))),
                "matchReason": item.get("matchReason", "Matched by semantic similarity and popularity."),
                "priority": "MEDIUM",
                "source": item.get("source", ["semantic_similarity"]),
            }
        )
    state["recommendations"] = fallback
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
    final_state = await app.ainvoke(state)
    return final_state
