from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from app.api.routes import recommendations
from app.api.routes import chat
from app.api.routes import reports
from app.config import settings
from app.observability import setup_observability
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
    return {"status": "healthy", "service": settings.APP_NAME}

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
    Base.metadata.create_all(bind=engine)
    logger.info("Database tables verified/created.")

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=settings.SERVER_PORT)
