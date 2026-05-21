import hashlib
import json
import logging
from typing import List

import redis.asyncio as redis
from app.config import settings
from sentence_transformers import SentenceTransformer

logger = logging.getLogger(__name__)


class EmbeddingService:
    def __init__(self) -> None:
        self._model = None
        self._redis = redis.from_url(settings.REDIS_URL, decode_responses=True)

    def _load_model(self) -> SentenceTransformer:
        if self._model is None:
            logger.info(f"Loading embedding model: {settings.EMBEDDING_MODEL_NAME}")
            self._model = SentenceTransformer(settings.EMBEDDING_MODEL_NAME)
        return self._model

    @staticmethod
    def _sanitize_text(text: str) -> str:
        cleaned = " ".join((text or "").split())
        return cleaned[:12000]

    @staticmethod
    def _cache_key(text: str, normalize: bool) -> str:
        raw = f"{settings.EMBEDDING_MODEL_NAME}:{normalize}:{text}".encode("utf-8")
        return f"embedding:v1:{hashlib.sha256(raw).hexdigest()}"

    async def embed_text(self, text: str, normalize: bool = True) -> List[float]:
        clean_text = self._sanitize_text(text)
        key = self._cache_key(clean_text, normalize)

        try:
            cached = await self._redis.get(key)
            if cached:
                return json.loads(cached)
        except Exception as exc:
            logger.warning(f"Embedding cache read failed: {exc}")

        model = self._load_model()
        vector = model.encode([clean_text], normalize_embeddings=normalize)[0].tolist()

        try:
            await self._redis.setex(
                key,
                settings.EMBEDDING_CACHE_TTL,
                json.dumps(vector),
            )
        except Exception as exc:
            logger.warning(f"Embedding cache write failed: {exc}")

        return vector

    async def embed_texts(self, texts: List[str], normalize: bool = True) -> List[List[float]]:
        if not texts:
            return []
        if len(texts) > 256:
            raise ValueError("Batch embedding limit is 256 texts per request.")

        clean_texts = [self._sanitize_text(t) for t in texts]
        vectors: List[List[float]] = []
        misses: List[str] = []
        miss_indices: List[int] = []

        for idx, text in enumerate(clean_texts):
            key = self._cache_key(text, normalize)
            try:
                cached = await self._redis.get(key)
                if cached:
                    vectors.append(json.loads(cached))
                    continue
            except Exception as exc:
                logger.warning(f"Embedding cache read failed: {exc}")
            vectors.append([])
            misses.append(text)
            miss_indices.append(idx)

        if misses:
            model = self._load_model()
            generated = model.encode(misses, normalize_embeddings=normalize).tolist()
            for i, vec in enumerate(generated):
                idx = miss_indices[i]
                vectors[idx] = vec
                key = self._cache_key(clean_texts[idx], normalize)
                try:
                    await self._redis.setex(
                        key,
                        settings.EMBEDDING_CACHE_TTL,
                        json.dumps(vec),
                    )
                except Exception as exc:
                    logger.warning(f"Embedding cache write failed: {exc}")

        return vectors


embedding_service = EmbeddingService()
