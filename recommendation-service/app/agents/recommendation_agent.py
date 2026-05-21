from typing import Dict, List

from app.agents import graph as graph_module
from opentelemetry import trace
import logging

logger = logging.getLogger(__name__)
tracer = trace.get_tracer(__name__)


class RecommendationAgent:
    async def recommend(self, user_id: str) -> Dict:
        with tracer.start_as_current_span("recommendation.agent.run") as span:
            span.set_attribute("recommendation.user_id", str(user_id))
            state = await graph_module.run_recommendation_graph(user_id)
            recommendations = state.get("recommendations", [])
            tool_trace = state.get("tool_trace", [])
            errors = state.get("errors", [])
            span.set_attribute("recommendation.tool_calls", len(tool_trace))
            span.set_attribute("recommendation.result_count", len(recommendations))
            span.set_attribute("recommendation.error_count", len(errors))
            logger.info(
                "recommendation_agent_completed",
                extra={
                    "user_id": str(user_id),
                    "tool_calls": len(tool_trace),
                    "result_count": len(recommendations),
                    "error_count": len(errors),
                },
            )
            return {
                "recommendations": recommendations,
                "reasoning_summary": state.get("reasoning_summary", ""),
                "tool_trace": tool_trace,
                "errors": errors,
            }


recommendation_agent = RecommendationAgent()
