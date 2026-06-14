from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import JSONResponse
from fastapi.middleware.cors import CORSMiddleware
import hashlib
import json
from app.api.routes import recommendations
from app.api.routes import chat
from app.api.routes import reports
from app.api.routes import courses
from app.config import settings
from app.observability import setup_observability
from app.utils.api_response import success_response
import asyncio
import logging

logger = logging.getLogger(__name__)

_COURSE_INDEX_FINGERPRINT_KEY = "startup:course-index:fingerprint"
_CLUSTERING_FINGERPRINT_KEY = "startup:clustering:fingerprint"

app = FastAPI(title=settings.APP_NAME)

# Initialize observability (Tracing, Metrics, Logger, Sentry)
setup_observability(app)

# Set up CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Include routers
app.include_router(recommendations.router, prefix="/api/v1/recommendations", tags=["recommendations"])
app.include_router(chat.router, prefix="/api/v1/chatbot", tags=["chatbot"])
app.include_router(reports.router, prefix="/api/v1/reports", tags=["reports"])
app.include_router(courses.router, prefix="/api/v1/courses", tags=["courses"])

@app.get("/health")
async def health_check():
    dependency_status = {"database": "unknown", "redis": "unknown", "vector_db": "unknown"}
    overall_healthy = True

    try:
        from sqlalchemy import text
        from app.models.database import engine

        with engine.connect() as connection:
            connection.execute(text("SELECT 1"))
        dependency_status["database"] = "healthy"
    except Exception as exc:
        dependency_status["database"] = f"unhealthy: {exc}"
        overall_healthy = False

    try:
        from app.services.recommendation_engine import redis_conn

        await redis_conn.ping()
        dependency_status["redis"] = "healthy"
    except Exception as exc:
        dependency_status["redis"] = f"unhealthy: {exc}"
        overall_healthy = False

    try:
        from app.retrieval.vector_store import vector_store

        await vector_store.client.collection_exists(collection_name=vector_store.courses_collection)
        dependency_status["vector_db"] = "healthy"
    except Exception as exc:
        dependency_status["vector_db"] = f"unhealthy: {exc}"
        overall_healthy = False

    return success_response(
        data={
            "status": "healthy" if overall_healthy else "degraded",
            "service": settings.APP_NAME,
            "dependencies": dependency_status,
        }
    )


@app.exception_handler(HTTPException)
async def http_exception_handler(request: Request, exc: HTTPException):
    if isinstance(exc.detail, dict) and {"success", "data", "error", "message"}.issubset(exc.detail.keys()):
        return JSONResponse(status_code=exc.status_code, content=exc.detail)
    return JSONResponse(
        status_code=exc.status_code,
        content={
            "success": False,
            "data": None,
            "error": {
                "code": "HTTP_ERROR",
                "message": str(exc.detail),
            },
            "message": str(exc.detail),
        },
    )

@app.get("/debug-sentry")
async def debug_sentry():
    1 / 0  # Trigger zero division error
    return {"message": "Should not reach here"}

@app.on_event("startup")
async def startup_event():
    logger.info(f"Starting {settings.APP_NAME}...")

    from app.models.database import engine, Base
    from app.models.recommendation import RecommendationHistory  # noqa: F401
    from app.models.chat import ChatSession, ChatMessage  # noqa: F401
    from app.models.report import StudentReport # noqa: F401
    from app.models.cluster import UserCluster, ClusterMetadata  # noqa: F401
    from app.models.search import SearchFeedbackAggregate, SearchFeedbackEvent, SearchQueryAnalytics  # noqa: F401
    Base.metadata.create_all(bind=engine)
    logger.info("Database tables verified/created.")

    asyncio.create_task(_index_courses_on_startup())
    asyncio.create_task(_periodic_course_reindex())


async def _index_courses_on_startup():
    try:
        from app.services.course_client import course_client
        from app.jobs.embedding_jobs import refresh_course_embeddings
        from app.retrieval.embedding_service import embedding_service
        from app.retrieval.vector_store import vector_store
        import redis.asyncio as aioredis

        await vector_store.ensure_collections(384)

        courses = await course_client.get_all_courses()
        if not courses:
            logger.warning("Startup indexing: no courses returned from courses-service")
            return
        logger.info(f"Startup indexing: fetched {len(courses)} courses")

        r = aioredis.from_url(settings.REDIS_URL, decode_responses=True)
        catalog_fingerprint = _build_course_catalog_fingerprint(courses)
        course_client.set_cached_courses(courses, catalog_fingerprint)
        previous_fingerprint = await r.get(_COURSE_INDEX_FINGERPRINT_KEY)

        if previous_fingerprint == catalog_fingerprint:
            logger.info("Startup indexing: course catalog unchanged, skipping full reindex")
        else:
            logger.info("Startup indexing: course catalog changed, refreshing embeddings incrementally...")
            await refresh_course_embeddings(courses)
            await r.set(_COURSE_INDEX_FINGERPRINT_KEY, catalog_fingerprint)

        # Flush stale retrieval caches so searches see the freshly indexed vectors
        try:
            if previous_fingerprint != catalog_fingerprint:
                keys = []
                for pattern in ("retrieval:v1:*", "recommendation:v2:*", "course-search:v1:*", "course-autocomplete:v1:*"):
                    keys.extend(await r.keys(pattern))
                if keys:
                    await r.delete(*keys)
                    logger.info(f"Flushed {len(keys)} stale recommendation/search cache entries")
        except Exception as exc:
            logger.warning(f"Recommendation cache flush failed: {exc}")
        finally:
            await r.aclose()

        try:
            embedding_service._load_model()
            logger.info("Startup indexing: embedding model warmed")
        except Exception as exc:
            logger.warning(f"Embedding warm-up failed: {exc}")
    except Exception as exc:
        logger.error(f"Startup course indexing failed: {exc}", exc_info=True)

    # Build user clusters so recommendations carry a collaborative-filtering signal
    await _run_clustering_on_startup()


async def _run_clustering_on_startup():
    try:
        from app.clustering.jobs import gather_known_user_ids, run_clustering_job_for_users
        import redis.asyncio as aioredis

        user_ids = gather_known_user_ids()
        if not user_ids:
            logger.info("Startup clustering: no known users yet, skipping")
            return
        fingerprint = _build_user_id_fingerprint(user_ids)
        r = aioredis.from_url(settings.REDIS_URL, decode_responses=True)
        previous_fingerprint = await r.get(_CLUSTERING_FINGERPRINT_KEY)
        if previous_fingerprint == fingerprint:
            logger.info("Startup clustering: known user set unchanged, skipping reclustering")
            await r.aclose()
            return

        logger.info(f"Startup clustering: clustering {len(user_ids)} known users...")
        result = await run_clustering_job_for_users(user_ids)
        await r.set(_CLUSTERING_FINGERPRINT_KEY, fingerprint)
        await r.aclose()
        logger.info(f"Startup clustering result: {result}")
    except Exception as exc:
        logger.error(f"Startup clustering failed: {exc}", exc_info=True)


def _build_course_catalog_fingerprint(courses: list[dict]) -> str:
    minimal_rows = []
    for course in sorted(courses, key=lambda item: str(item.get("id"))):
        minimal_rows.append(
            {
                "id": str(course.get("id")),
                "title": course.get("title"),
                "description": course.get("description"),
                "subjectName": course.get("subjectName"),
                "teacherName": course.get("teacherName"),
                "enrollmentCount": course.get("enrollmentCount"),
                "updatedAt": course.get("updatedAt"),
            }
        )
    raw = json.dumps(minimal_rows, sort_keys=True, default=str).encode("utf-8")
    return hashlib.sha256(raw).hexdigest()


def _build_user_id_fingerprint(user_ids: list[str]) -> str:
    raw = json.dumps(sorted(str(user_id) for user_id in user_ids)).encode("utf-8")
    return hashlib.sha256(raw).hexdigest()

async def _periodic_course_reindex():
    """Periodically re-index courses so new ones created via the dashboard
    become searchable without requiring a service restart."""
    interval_seconds = settings.EMBEDDING_REFRESH_INTERVAL_MINUTES * 60
    logger.info(f"Periodic course re-index: will run every {settings.EMBEDDING_REFRESH_INTERVAL_MINUTES} minutes")

    # Wait for the initial startup indexing to complete first
    await asyncio.sleep(interval_seconds)

    while True:
        try:
            from app.services.course_client import course_client
            from app.jobs.embedding_jobs import refresh_course_embeddings
            import redis.asyncio as aioredis

            # Force-clear the in-memory cache so get_all_courses() fetches live data
            course_client.clear_cached_courses()
            courses = await course_client.get_all_courses()
            if not courses:
                logger.warning("Periodic reindex: no courses returned")
                await asyncio.sleep(interval_seconds)
                continue

            catalog_fingerprint = _build_course_catalog_fingerprint(courses)
            previous_fingerprint = course_client.get_catalog_fingerprint()

            if previous_fingerprint == catalog_fingerprint:
                logger.info(f"Periodic reindex: catalog unchanged ({len(courses)} courses)")
            else:
                logger.info(f"Periodic reindex: catalog changed, re-indexing {len(courses)} courses...")
                await refresh_course_embeddings(courses)
                course_client.set_cached_courses(courses, catalog_fingerprint)

                r = aioredis.from_url(settings.REDIS_URL, decode_responses=True)
                await r.set(_COURSE_INDEX_FINGERPRINT_KEY, catalog_fingerprint)
                # Flush stale search caches
                keys = []
                for pattern in ("retrieval:v1:*", "recommendation:v2:*", "course-search:v2:*", "course-autocomplete:v2:*"):
                    keys.extend(await r.keys(pattern))
                if keys:
                    await r.delete(*keys)
                    logger.info(f"Periodic reindex: flushed {len(keys)} stale cache entries")
                await r.aclose()
                logger.info("Periodic reindex: completed successfully")
        except Exception as exc:
            logger.error(f"Periodic reindex failed: {exc}", exc_info=True)

        await asyncio.sleep(interval_seconds)


@app.post("/api/v1/courses/reindex")
async def trigger_course_reindex(request: Request):
    """Event-driven endpoint: courses-service calls this after course create/update
    to force the recommendation service to immediately refresh its catalog."""
    secret = request.headers.get("x-internal-service-secret", "")
    expected = settings.INTERNAL_SERVICE_SECRET
    if secret != expected:
        raise HTTPException(status_code=403, detail="Forbidden")

    try:
        from app.services.course_client import course_client
        from app.jobs.embedding_jobs import refresh_course_embeddings
        import redis.asyncio as aioredis

        course_client.clear_cached_courses()
        courses = await course_client.get_all_courses()
        if not courses:
            return JSONResponse(content={"success": True, "message": "No courses found", "indexed": 0})

        catalog_fingerprint = _build_course_catalog_fingerprint(courses)
        await refresh_course_embeddings(courses)
        course_client.set_cached_courses(courses, catalog_fingerprint)

        r = aioredis.from_url(settings.REDIS_URL, decode_responses=True)
        await r.set(_COURSE_INDEX_FINGERPRINT_KEY, catalog_fingerprint)
        keys = []
        for pattern in ("retrieval:v1:*", "course-search:v2:*", "course-autocomplete:v2:*"):
            keys.extend(await r.keys(pattern))
        if keys:
            await r.delete(*keys)
        await r.aclose()

        logger.info(f"Event-driven reindex: indexed {len(courses)} courses")
        return JSONResponse(content={"success": True, "message": "Reindex complete", "indexed": len(courses)})
    except Exception as exc:
        logger.error(f"Event-driven reindex failed: {exc}", exc_info=True)
        raise HTTPException(status_code=500, detail=str(exc))


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=settings.SERVER_PORT)
