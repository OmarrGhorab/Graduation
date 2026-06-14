from pydantic_settings import BaseSettings, SettingsConfigDict

class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", extra="ignore")

    APP_NAME: str = "AI Course Recommendation Service"
    SERVER_PORT: int = 8095
    
    # AI Config
    AI_API_KEY: str
    AI_MODEL: str = "gpt-5.4-mini"
    AI_BASE_URL: str = "https://api.freemodel.dev"
    AI_WIRE_API: str = "responses"
    AI_REASONING_EFFORT: str = "medium"
    DISABLE_RESPONSE_STORAGE: bool = True
    OPENROUTER_SITE_URL: str = "http://localhost:3000"
    OPENROUTER_APP_NAME: str = "Graduation Platform"
    
    # Databases
    DATABASE_URL: str
    REDIS_URL: str = "redis://localhost:6379"
    VECTOR_DB_URL: str = "http://localhost:6333"
    
    # Internal Communication
    INTERNAL_SERVICE_SECRET: str
    COURSES_SERVICE_URL: str = "http://localhost:8085"
    AUTH_SERVICE_URL: str = "http://localhost:6001"
    NOTIFICATION_SERVICE_URL: str = "http://localhost:6003"
    
    # Cache settings
    RECOMMENDATION_CACHE_TTL: int = 21600  # 6 hours in seconds
    TRENDING_CACHE_TTL: int = 3600        # 1 hour in seconds
    AGENT_RECOMMENDATIONS_ENABLED: bool = False
    AGENT_MAX_TOOL_CALLS: int = 8
    AGENT_TOP_K_CANDIDATES: int = 20
    AGENT_FINAL_RECOMMENDATION_COUNT: int = 6
    AGENT_REASONING_LOG_TTL: int = 2592000  # 30 days in seconds

    # Retrieval and embeddings
    EMBEDDING_MODEL_NAME: str = "BAAI/bge-small-en-v1.5"
    EMBEDDING_CACHE_TTL: int = 604800      # 7 days in seconds
    EMBEDDING_REFRESH_INTERVAL_MINUTES: int = 30
    RETRIEVAL_TOP_K: int = 20
    RETRIEVAL_CACHE_TTL: int = 1800        # 30 minutes in seconds
    CATALOG_CACHE_TTL: int = 300           # 5 minutes – how long the in-memory course list is trusted
    USER_PROFILE_CACHE_TTL: int = 5        # 5 seconds – how long a cached user analytics profile is trusted
    QDRANT_COURSES_COLLECTION: str = "courses"
    QDRANT_USERS_COLLECTION: str = "users"
    QDRANT_CLUSTERS_COLLECTION: str = "clusters"

    # Clustering
    CLUSTER_COUNT: int = 8
    CLUSTER_REFRESH_INTERVAL_MINUTES: int = 60
    CLUSTER_CACHE_TTL: int = 3600

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

settings = Settings()
