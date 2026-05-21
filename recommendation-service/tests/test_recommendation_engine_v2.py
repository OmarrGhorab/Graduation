import json

import pytest

from app.config import settings
from app.services import recommendation_engine as engine


class FakeRedis:
    def __init__(self):
        self.store = {}

    async def get(self, key):
        return self.store.get(key)

    async def setex(self, key, ttl, value):
        self.store[key] = value

    async def delete(self, key):
        self.store.pop(key, None)


class FakeRecommendationAgent:
    async def recommend(self, user_id):
        return {
            "recommendations": [
                {"courseId": "c1", "score": 80, "matchReason": "reason", "priority": "HIGH"}
            ],
            "reasoning_summary": "summary",
            "tool_trace": [{"tool": "search_relevant_courses"}],
            "errors": [],
        }


@pytest.mark.asyncio
async def test_agentic_recommendation_path_uses_v2_cache(monkeypatch):
    fake_redis = FakeRedis()
    monkeypatch.setattr(engine, "redis_conn", fake_redis)
    monkeypatch.setattr(engine, "recommendation_agent", FakeRecommendationAgent())
    monkeypatch.setattr(settings, "AGENT_RECOMMENDATIONS_ENABLED", True)
    monkeypatch.setattr(engine, "_persist_v2_recommendation_history", lambda *args, **kwargs: None)

    result = await engine.get_personalized_recommendations("user-1")
    assert result[0]["courseId"] == "c1"
    assert json.loads(fake_redis.store["recommendation:v2:user-1"])[0]["courseId"] == "c1"


@pytest.mark.asyncio
async def test_explanation_roundtrip(monkeypatch):
    fake_redis = FakeRedis()
    fake_redis.store["recommendation:v2:explain:user-1"] = json.dumps({"reasoningSummary": "ok", "toolTrace": [], "errors": []})
    monkeypatch.setattr(engine, "redis_conn", fake_redis)

    explanation = await engine.get_recommendation_explanation("user-1")
    assert explanation["reasoningSummary"] == "ok"
