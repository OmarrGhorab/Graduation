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

    class Config:
        env_file = ".env"

settings = Settings()
