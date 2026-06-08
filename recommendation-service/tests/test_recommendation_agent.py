import pytest

from app.agents import graph as graph_module
from app.agents.recommendation_agent import RecommendationAgent


@pytest.mark.asyncio
async def test_recommendation_agent_returns_structured_result(monkeypatch):
    async def fake_run(user_id):
        return {
            "recommendations": [{"courseId": "1", "score": 90, "matchReason": "x", "priority": "HIGH"}],
            "reasoning_summary": "summary",
            "tool_trace": [{"tool": "get_user_profile"}],
            "errors": [],
        }

    monkeypatch.setattr(graph_module, "run_recommendation_graph", fake_run)

    agent = RecommendationAgent()
    result = await agent.recommend("user-1")
    assert result["recommendations"][0]["courseId"] == "1"
    assert result["reasoning_summary"] == "summary"
