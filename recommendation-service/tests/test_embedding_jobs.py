import pytest

from app.jobs import embedding_jobs


class FakeEmbeddingService:
    async def embed_text(self, text, normalize=True):
        return [0.1, 0.2, 0.3]


class FakeVectorStore:
    def __init__(self):
        self.upserted = None

    async def upsert_user_vector(self, user_id, vector, payload):
        self.upserted = {
            "user_id": user_id,
            "vector": vector,
            "payload": payload,
        }


@pytest.mark.asyncio
async def test_refresh_user_embeddings_upserts_user_vector(monkeypatch):
    fake_vector_store = FakeVectorStore()
    monkeypatch.setattr(embedding_jobs, "embedding_service", FakeEmbeddingService())
    monkeypatch.setattr(embedding_jobs, "vector_store", fake_vector_store)

    document = await embedding_jobs.refresh_user_embeddings("user-1", {"UserInterests": ["backend"]})

    assert document["entity_type"] == "user"
    assert document["entity_id"] == "user-1"
    assert fake_vector_store.upserted["user_id"] == "user-1"
    assert fake_vector_store.upserted["payload"]["entity_type"] == "user"
