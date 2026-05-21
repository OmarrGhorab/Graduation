from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import JSONResponse
from fastapi.middleware.cors import CORSMiddleware
from app.api.routes import recommendations
from app.api.routes import chat
from app.api.routes import reports
from app.config import settings
from app.observability import setup_observability
from app.utils.api_response import success_response
import logging

logger = logging.getLogger(__name__)

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

    # Auto-create new tables (chat_sessions, chat_messages) if they don't exist
    from app.models.database import engine, Base
    from app.models.recommendation import RecommendationHistory  # noqa: F401
    from app.models.chat import ChatSession, ChatMessage  # noqa: F401
    from app.models.report import StudentReport # noqa: F401
    from app.models.cluster import UserCluster, ClusterMetadata  # noqa: F401
    Base.metadata.create_all(bind=engine)
    logger.info("Database tables verified/created.")

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=settings.SERVER_PORT)
