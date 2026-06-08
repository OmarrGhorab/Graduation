from typing import Any, Dict, List, Optional, TypedDict


class RecommendationState(TypedDict):
    user_id: str
    tool_calls: int
    done: bool
    next_tool: Optional[str]
    next_tool_args: Dict[str, Any]
    tool_trace: List[Dict[str, Any]]
    context: Dict[str, Any]
    candidates: List[Dict[str, Any]]
    recommendations: List[Dict[str, Any]]
    errors: List[str]
    reasoning_summary: str
