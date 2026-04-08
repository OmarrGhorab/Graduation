from pydantic_settings import BaseSettings
from typing import Optional

class Settings(BaseSettings):
    APP_NAME: str = "AI Course Recommendation Service"
    SERVER_PORT: int = 8095
    
    # AI Config
    AI_API_KEY: str
    AI_MODEL: str = "gemma-4-26b"
    
    # Databases
    DATABASE_URL: str
    REDIS_URL: str = "redis://localhost:6379"
    
    # Internal Communication
    INTERNAL_SERVICE_SECRET: str
    COURSES_SERVICE_URL: str = "http://localhost:8085"
    AUTH_SERVICE_URL: str = "http://localhost:6001"
    
    # Cache settings
    RECOMMENDATION_CACHE_TTL: int = 21600  # 6 hours in seconds

    # Chatbot Config
    CHATBOT_MAX_CONTEXT_MESSAGES: int = 20    # messages sent to AI per request
    CHATBOT_MAX_HISTORY_PER_CHAT: int = 100   # max stored messages per chat
    CHATBOT_MAX_ACTIVE_CHATS: int = 10        # max open chats per user
    CHATBOT_MAX_MESSAGE_LENGTH: int = 2000    # max characters per message
    CHATBOT_COURSE_CONTEXT_TTL: int = 3600    # 1 hour cache for course data
    
    # Cloudinary
    CLOUDINARY_CLOUD_NAME: str
    CLOUDINARY_API_KEY: str
    CLOUDINARY_API_SECRET: str
    CLOUDINARY_FOLDER: str = "chatbot-media"

    class Config:
        env_file = ".env"

settings = Settings()
