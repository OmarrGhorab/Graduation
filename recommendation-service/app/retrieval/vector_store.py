import logging
from typing import Dict, List, Optional

from app.config import settings
from qdrant_client import AsyncQdrantClient
from qdrant_client.http import models
from opentelemetry import trace
from time import perf_counter

logger = logging.getLogger(__name__)
tracer = trace.get_tracer(__name__)


class VectorStore:
    def __init__(self) -> None:
        self.client = AsyncQdrantClient(url=settings.VECTOR_DB_URL, timeout=10.0)
        self.courses_collection = settings.QDRANT_COURSES_COLLECTION
        self.users_collection = settings.QDRANT_USERS_COLLECTION
        self.clusters_collection = settings.QDRANT_CLUSTERS_COLLECTION

    async def ensure_collections(self, vector_size: int) -> None:
        for collection_name in [
            self.courses_collection,
            self.users_collection,
            self.clusters_collection,
        ]:
            exists = await self.client.collection_exists(collection_name=collection_name)
            if exists:
                continue
            await self.client.create_collection(
                collection_name=collection_name,
                vectors_config=models.VectorParams(
                    size=vector_size,
                    distance=models.Distance.COSINE,
                ),
            )
            logger.info(f"Created Qdrant collection: {collection_name}")

    async def upsert_course_vector(
        self,
        course_id: str,
        vector: List[float],
        payload: Dict,
    ) -> None:
        await self.client.upsert(
            collection_name=self.courses_collection,
            points=[
                models.PointStruct(
                    id=str(course_id),
                    vector=vector,
                    payload=payload,
                )
            ],
        )

    async def upsert_user_vector(
        self,
        user_id: str,
        vector: List[float],
        payload: Dict,
    ) -> None:
        await self.client.upsert(
            collection_name=self.users_collection,
            points=[
                models.PointStruct(
                    id=str(user_id),
                    vector=vector,
                    payload=payload,
                )
            ],
        )

    async def upsert_cluster_vector(
        self,
        cluster_id: str,
        vector: List[float],
        payload: Dict,
    ) -> None:
        await self.client.upsert(
            collection_name=self.clusters_collection,
            points=[
                models.PointStruct(
                    id=int(cluster_id),
                    vector=vector,
                    payload=payload,
                )
            ],
        )

    async def search_courses(
        self,
        query_vector: List[float],
        top_k: Optional[int] = None,
        filters: Optional[Dict] = None,
    ) -> List[Dict]:
        with tracer.start_as_current_span("recommendation.vector.search") as span:
            start = perf_counter()
            limit = top_k or settings.RETRIEVAL_TOP_K
            query_filter = None

            if filters:
                conditions: List[models.FieldCondition] = []
                for key, value in filters.items():
                    conditions.append(
                        models.FieldCondition(
                            key=key,
                            match=models.MatchValue(value=value),
                        )
                    )
                query_filter = models.Filter(must=conditions)

            results = await self.client.search(
                collection_name=self.courses_collection,
                query_vector=query_vector,
                query_filter=query_filter,
                limit=limit,
                with_payload=True,
            )

            parsed: List[Dict] = []
            for item in results:
                parsed.append(
                    {
                        "courseId": str(item.id),
                        "similarityScore": float(item.score),
                        "payload": item.payload or {},
                    }
                )

            duration_ms = (perf_counter() - start) * 1000
            span.set_attribute("vector.collection", self.courses_collection)
            span.set_attribute("vector.limit", limit)
            span.set_attribute("vector.result_count", len(parsed))
            span.set_attribute("vector.duration_ms", duration_ms)
            logger.info(
                "vector_search_timing",
                extra={
                    "collection": self.courses_collection,
                    "limit": limit,
                    "result_count": len(parsed),
                    "duration_ms": round(duration_ms, 2),
                },
            )
            return parsed


vector_store = VectorStore()
