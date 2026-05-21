from typing import Dict, List

from app.agents.graph import run_recommendation_graph


class RecommendationAgent:
    async def recommend(self, user_id: str) -> Dict:
        state = await run_recommendation_graph(user_id)
        return {
            "recommendations": state.get("recommendations", []),
            "reasoning_summary": state.get("reasoning_summary", ""),
            "tool_trace": state.get("tool_trace", []),
            "errors": state.get("errors", []),
        }


recommendation_agent = RecommendationAgent()
