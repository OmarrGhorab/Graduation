from fastapi import APIRouter, Depends, HTTPException, BackgroundTasks
from app.services.recommendation_engine import get_personalized_recommendations
from app.api.dependencies import get_current_user
from typing import List, Dict
import logging

router = APIRouter()
logger = logging.getLogger(__name__)

@router.get("/")
async def get_my_recommendations(user = Depends(get_current_user)):
    """
    Returns a list of AI-powered course recommendations for the authenticated user.
    """
    user_id = user["user_id"]
    try:
        recommendations = await get_personalized_recommendations(user_id)
        return {
            "success": True,
            "data": recommendations
        }
    except Exception as e:
        logger.error(f"Failed to get recommendations for {user_id}: {str(e)}")
        raise HTTPException(status_code=500, detail="Could not generate recommendations")

@router.post("/refresh")
async def refresh_recommendations(user = Depends(get_current_user)):
    """
    Invalidates the cache and forces a fresh recommendation generation.
    """
    # In a real app, you might trigger this via BackgroundTasks
    # For now, we'll just allow the next GET to be a miss if we manually delete the key
    from app.services.recommendation_engine import redis_conn
    cache_key = f"recommendation:v1:{user['user_id']}"
    await redis_conn.delete(cache_key)
    
    return {"success": True, "message": "Recommendations cache cleared"}

@router.delete("/cache/{user_id}")
async def invalidate_cache(user_id: str):
    """
    Called by other services when a user's data changes.
    Invalidating the cache forces the next visit to be a fresh AI calculation.
    """
    from app.services.recommendation_engine import recommendation_engine
    await recommendation_engine.clear_cache(user_id)
    return {"success": True, "message": f"Cache invalidated for user {user_id}"}

@router.post("/test")
async def chat_test(data: Dict):
    """
    Directly test the AI connectivity with a message.
    Body: {"message": "Hi!"}
    """
    from app.services.gemma_client import gemma_client
    message = data.get("message", "Hello!")
    response = await gemma_client.chat(message)
    return {
        "success": True,
        "response": response
    }

@router.get("/trending")
async def get_trending_courses():
    """
    Fallback endpoint for non-personalized trending courses.
    """
    # This could call a simplified version of the engine or just hardcoded logic
    return {
        "success": True,
        "data": [],
        "note": "Non-personalized trending logic to be implemented"
    }
