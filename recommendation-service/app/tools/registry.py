import re
from typing import Any, Awaitable, Callable, Dict

from pydantic import BaseModel, ValidationError
from opentelemetry import trace
import logging

from app.tools.cluster_tools import (
    get_cluster_top_courses_tool,
    get_similar_users_tool,
    get_user_cluster_tool,
)
from app.tools.course_tools import get_trending_courses_tool, search_relevant_courses_tool
from app.tools.schemas import (
    GetClusterTopCoursesInput,
    GetClusterTopCoursesOutput,
    GetRecentCartActivityInput,
    GetRecentCartActivityOutput,
    GetSimilarUsersInput,
    GetSimilarUsersOutput,
    GetTrendingCoursesInput,
    GetTrendingCoursesOutput,
    GetUserClusterInput,
    GetUserClusterOutput,
    GetUserHistoryInput,
    GetUserHistoryOutput,
    GetUserProfileInput,
    GetUserProfileOutput,
    SearchRelevantCoursesInput,
    SearchRelevantCoursesOutput,
    ToolDefinition,
    ToolInvokeResponse,
)
from app.tools.user_tools import (
    get_recent_cart_activity_tool,
    get_user_history_tool,
    get_user_profile_tool,
)


ToolHandler = Callable[..., Awaitable[Dict[str, Any]]]
logger = logging.getLogger(__name__)
tracer = trace.get_tracer(__name__)


def _sanitize_text(value: str) -> str:
    if not value:
        return value
    cleaned = re.sub(r"(?i)(ignore previous instructions|system prompt|developer message)", "[redacted]", value)
    return cleaned[:400]


def _truncate_obj(value: Any, max_items: int = 25) -> Any:
    if isinstance(value, str):
        return _sanitize_text(value)
    if isinstance(value, list):
        truncated = value[:max_items]
        return [_truncate_obj(v, max_items=max_items) for v in truncated]
    if isinstance(value, dict):
        keys = list(value.keys())[:max_items]
        return {k: _truncate_obj(value[k], max_items=max_items) for k in keys}
    return value


def _sanitize_payload(value: Any, max_items: int = 25) -> Any:
    return _truncate_obj(value, max_items=max_items)


class ToolRegistry:
    def __init__(self) -> None:
        self._definitions: Dict[str, ToolDefinition] = {}
        self._handlers: Dict[str, ToolHandler] = {}
        self._register_defaults()

    def _register(
        self,
        name: str,
        description: str,
        input_model: type[BaseModel],
        output_model: type[BaseModel],
        handler: ToolHandler,
    ) -> None:
        self._definitions[name] = ToolDefinition(
            name=name,
            description=description,
            input_model=input_model,
            output_model=output_model,
        )
        self._handlers[name] = handler

    def _register_defaults(self) -> None:
        self._register(
            name="get_user_profile",
            description="Fetch user analytics profile for recommendation context.",
            input_model=GetUserProfileInput,
            output_model=GetUserProfileOutput,
            handler=get_user_profile_tool,
        )
        self._register(
            name="get_user_history",
            description="Fetch user enrollment and interest history.",
            input_model=GetUserHistoryInput,
            output_model=GetUserHistoryOutput,
            handler=get_user_history_tool,
        )
        self._register(
            name="get_recent_cart_activity",
            description="Fetch recent cart subjects for a user.",
            input_model=GetRecentCartActivityInput,
            output_model=GetRecentCartActivityOutput,
            handler=get_recent_cart_activity_tool,
        )
        self._register(
            name="search_relevant_courses",
            description="Search relevant courses using semantic retrieval and hybrid ranking.",
            input_model=SearchRelevantCoursesInput,
            output_model=SearchRelevantCoursesOutput,
            handler=search_relevant_courses_tool,
        )
        self._register(
            name="get_trending_courses",
            description="Fetch globally trending courses.",
            input_model=GetTrendingCoursesInput,
            output_model=GetTrendingCoursesOutput,
            handler=get_trending_courses_tool,
        )
        self._register(
            name="get_user_cluster",
            description="Fetch cluster assignment for a specific user.",
            input_model=GetUserClusterInput,
            output_model=GetUserClusterOutput,
            handler=get_user_cluster_tool,
        )
        self._register(
            name="get_similar_users",
            description="Fetch users similar by cluster membership.",
            input_model=GetSimilarUsersInput,
            output_model=GetSimilarUsersOutput,
            handler=get_similar_users_tool,
        )
        self._register(
            name="get_cluster_top_courses",
            description="Fetch top courses for a cluster.",
            input_model=GetClusterTopCoursesInput,
            output_model=GetClusterTopCoursesOutput,
            handler=get_cluster_top_courses_tool,
        )

    def list_tools(self) -> Dict[str, Dict[str, Any]]:
        result: Dict[str, Dict[str, Any]] = {}
        for name, definition in self._definitions.items():
            result[name] = {
                "name": definition.name,
                "description": definition.description,
                "input_schema": definition.input_model.model_json_schema(),
                "output_schema": definition.output_model.model_json_schema(),
            }
        return result

    async def invoke(self, name: str, arguments: Dict[str, Any]) -> ToolInvokeResponse:
        with tracer.start_as_current_span("recommendation.tool.execute") as span:
            span.set_attribute("tool.name", name)
            span.set_attribute("tool.arg_keys", ",".join(sorted(arguments.keys())))
            if name not in self._definitions:
                logger.warning("tool_unknown", extra={"tool_name": name})
                span.set_attribute("tool.success", False)
                return ToolInvokeResponse(name=name, success=False, data={}, error=f"Unknown tool: {name}")

            definition = self._definitions[name]
            handler = self._handlers[name]

            try:
                validated_input = definition.input_model(**arguments)
            except ValidationError as exc:
                logger.warning("tool_invalid_input", extra={"tool_name": name, "error": str(exc)})
                span.set_attribute("tool.success", False)
                return ToolInvokeResponse(name=name, success=False, data={}, error=f"Invalid input: {exc}")

            try:
                raw_output = await handler(**validated_input.model_dump())
                safe_output = _truncate_obj(raw_output, max_items=25)
                validated_output = definition.output_model(**safe_output)
                data = validated_output.model_dump()
                logger.info(
                    "tool_executed",
                    extra={
                        "tool_name": name,
                        "success": True,
                        "result_keys": list(data.keys())[:10],
                    },
                )
                span.set_attribute("tool.success", True)
                span.set_attribute("tool.result_keys", ",".join(list(data.keys())[:10]))
                return ToolInvokeResponse(
                    name=name,
                    success=True,
                    data=data,
                    error=None,
                )
            except ValidationError as exc:
                logger.warning("tool_invalid_output", extra={"tool_name": name, "error": str(exc)})
                span.set_attribute("tool.success", False)
                return ToolInvokeResponse(name=name, success=False, data={}, error=f"Invalid output: {exc}")
            except Exception as exc:
                logger.error("tool_execution_failed", extra={"tool_name": name, "error": str(exc)})
                span.set_attribute("tool.success", False)
                return ToolInvokeResponse(name=name, success=False, data={}, error=str(exc))


tool_registry = ToolRegistry()
