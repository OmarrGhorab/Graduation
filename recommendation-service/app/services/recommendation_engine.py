from app.services.course_client import course_client
from app.services.gemma_client import gemma_client
from app.utils.prompt_builder import build_recommendation_prompt
from app.config import settings
import logging
import json
import redis.asyncio as redis

logger = logging.getLogger(__name__)

# Initialize Redis
redis_conn = redis.from_url(settings.REDIS_URL, decode_responses=True)

async def get_personalized_recommendations(user_id: str):
    """
    Orchestrates the full recommendation flow:
    1. Check cache
    2. Fetch user analytics + course catalog
    3. Generate AI prompt
    4. Call Gemma 4
    5. Cache and return results
    """
    
    # 1. Check Cache
    cache_key = f"recommendation:v1:{user_id}"
    try:
        cached_data = await redis_conn.get(cache_key)
        if cached_data:
            logger.info(f"Cache hit for user {user_id}")
            return json.loads(cached_data)
    except Exception as e:
        logger.warning(f"Redis error: {str(e)}")

    # 2. Fetch Data
    logger.info(f"Cache miss for user {user_id}. Fetching fresh data...")
    try:
        user_profile = await course_client.get_user_analytics_profile(user_id)
        logger.info(f"Fetched user profile: {user_profile is not None}")
        
        all_courses = await course_client.get_all_courses()
        logger.info(f"Fetched {len(all_courses) if all_courses else 0} courses")
        
        if not all_courses:
            logger.warning("No courses available for recommendation")
            return []

        # 3. Build Prompt
        logger.info("Building AI prompt...")
        prompt = build_recommendation_prompt(user_profile, all_courses)
        logger.info("Prompt built successfully")
        
        # 4. Generate with Gemma 4
        logger.info(f"Calling AI model: {settings.AI_MODEL}")
        recommendations = await gemma_client.generate_recommendations(prompt)
        logger.info(f"AI returned {len(recommendations) if isinstance(recommendations, list) else 'invalid'} items")
        
        # 5. Enhance data (Optional: merge full course details back if AI only returned IDs)
        # For now, we assume AI returns a good amount of info as per prompt
        
        # 6. Cache Results
        try:
            if isinstance(recommendations, list):
                await redis_conn.setex(
                    cache_key,
                    settings.RECOMMENDATION_CACHE_TTL,
                    json.dumps(recommendations)
                )
        except Exception as e:
            logger.warning(f"Failed to cache recommendations: {str(e)}")

        return recommendations
    except Exception as e:
        logger.error(f"CRITICAL: recommendation_engine failure: {str(e)}", exc_info=True)
        raise

async def clear_cache(user_id: str):
    """Removes the cached recommendations for a specific user."""
    cache_key = f"recommendation:v1:{user_id}"
    await redis_conn.delete(cache_key)
    logger.info(f"Cache cleared for user: {user_id}")
