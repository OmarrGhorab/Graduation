import pytest

from app.retrieval.vector_store import VectorStore


class FakeSearchItem:
    def __init__(self, item_id, score, payload):
        self.id = item_id
        self.score = score
        self.payload = payload


class FakeClient:
    def __init__(self):
        self.search_called = False

    async def collection_exists(self, collection_name):
        return True

    async def create_collection(self, **kwargs):
        return None

    async def upsert(self, **kwargs):
        return None

    async def search(self, **kwargs):
        self.search_called = True
        return [FakeSearchItem("1", 0.88, {"title": "Test Course"})]


@pytest.mark.asyncio
async def test_vector_store_search_courses(monkeypatch):
    store = VectorStore()
    store.client = FakeClient()

    results = await store.search_courses([0.1, 0.2, 0.3], top_k=1)

    assert store.client.search_called is True
    assert results[0]["courseId"] == "1"
    assert results[0]["payload"]["title"] == "Test Course"
