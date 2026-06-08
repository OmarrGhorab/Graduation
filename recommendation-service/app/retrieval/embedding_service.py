import hashlib
import json
import logging
from typing import List

import redis.asyncio as redis
from app.config import settings
from sentence_transformers import SentenceTransformer
from opentelemetry import trace
from time import perf_counter

logger = logging.getLogger(__name__)
tracer = trace.get_tracer(__name__)


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
        with tracer.start_as_current_span("recommendation.embedding.generate") as span:
            start = perf_counter()
            clean_text = self._sanitize_text(text)
            key = self._cache_key(clean_text, normalize)
            span.set_attribute("embedding.model", settings.EMBEDDING_MODEL_NAME)
            span.set_attribute("embedding.batch_size", 1)
            span.set_attribute("embedding.input_length", len(clean_text))

            try:
                cached = await self._redis.get(key)
                if cached:
                    span.set_attribute("embedding.cache_hit", True)
                    return json.loads(cached)
            except Exception as exc:
                logger.warning(f"Embedding cache read failed: {exc}")

            span.set_attribute("embedding.cache_hit", False)
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

            span.set_attribute("embedding.duration_ms", (perf_counter() - start) * 1000)
            return vector

    async def embed_texts(self, texts: List[str], normalize: bool = True) -> List[List[float]]:
        with tracer.start_as_current_span("recommendation.embedding.generate") as span:
            start = perf_counter()
            span.set_attribute("embedding.model", settings.EMBEDDING_MODEL_NAME)
            span.set_attribute("embedding.batch_size", len(texts))
            if not texts:
                span.set_attribute("embedding.duration_ms", 0)
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

            span.set_attribute("embedding.cache_hits", len(texts) - len(misses))
            span.set_attribute("embedding.cache_misses", len(misses))
            span.set_attribute("embedding.duration_ms", (perf_counter() - start) * 1000)
            return vectors


embedding_service = EmbeddingService()
