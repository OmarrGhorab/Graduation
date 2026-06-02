import asyncio
import logging
from typing import List

from app.config import settings
from app.retrieval.course_indexer import build_course_embedding_text, build_course_embedding_payload
from app.retrieval.embedding_service import embedding_service
from app.retrieval.vector_store import vector_store

logger = logging.getLogger(__name__)


async def refresh_course_embeddings(courses: List[dict]) -> List[dict]:
    if not courses:
        return []

    texts = [build_course_embedding_text(course) for course in courses]
    vectors = await embedding_service.embed_texts(texts, normalize=True)

    results: List[dict] = []
    for course, vector in zip(courses, vectors):
        if not vector:
            continue
        course_id = str(course.get("id"))
        payload = build_course_embedding_payload(course)
        try:
            await vector_store.upsert_course_vector(course_id, vector, payload)
        except Exception as exc:
            logger.warning(f"Failed to upsert vector for course {course_id}: {exc}")
            continue
        results.append({
            "entity_type": "course",
            "entity_id": course_id,
            "vector": vector,
            "metadata": course,
        })
    logger.info(f"Indexed {len(results)}/{len(courses)} course vectors into Qdrant")
    return results


async def refresh_user_embeddings(user_id: str, user_profile: dict) -> dict:
    from app.clustering.feature_builder import build_user_behavior_summary

    text = build_user_behavior_summary(user_id, user_profile)
    vector = await embedding_service.embed_text(text, normalize=True)
    document = {
        "entity_type": "user",
        "entity_id": str(user_id),
        "vector": vector,
        "metadata": user_profile,
    }
    await vector_store.upsert_user_vector(
        str(user_id),
        vector,
        {
            "entity_type": "user",
            "entity_id": str(user_id),
            "summary": text[:1000],
        },
    )
    return document


async def refresh_embeddings_background(courses: List[dict]) -> None:
    try:
        await refresh_course_embeddings(courses)
        logger.info("Embedding refresh job completed for courses")
    except Exception as exc:
        logger.error(f"Embedding refresh job failed: {exc}")
