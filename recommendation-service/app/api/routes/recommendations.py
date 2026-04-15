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
async def refresh_recommendations(background_tasks: BackgroundTasks, user = Depends(get_current_user)):
    """
    Invalidates the cache and triggers a fresh recommendation generation in the background.
    """
    from app.services.recommendation_engine import clear_cache, get_personalized_recommendations
    user_id = user["user_id"]
    
    # 1. Clear current cache
    await clear_cache(user_id)
    
    # 2. Trigger fresh generation in background
    background_tasks.add_task(get_personalized_recommendations, user_id)
    
    logger.info(f"Public background refresh requested for user {user_id}")
    return {"success": True, "message": "Recommendations cache cleared and refresh started in background"}

@router.delete("/cache/{user_id}")
async def invalidate_cache(user_id: str, background_tasks: BackgroundTasks):
    """
    Called by other services when a user's data changes.
    Invalidating the cache AND pre-triggering the next AI calculation in the background.
    """
    from app.services.recommendation_engine import clear_cache, get_personalized_recommendations
    
    # 1. Clear current cache
    await clear_cache(user_id)
    
    # 2. Trigger fresh generation in background
    background_tasks.add_task(get_personalized_recommendations, user_id)
    
    logger.info(f"Background refresh triggered for user {user_id}")
    return {"success": True, "message": f"Cache invalidated and background refresh started for user {user_id}"}

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
    Returns globally trending courses based on popularity and authority.
    """
    from app.services.recommendation_engine import get_trending_recommendations
    try:
        trending = await get_trending_recommendations()
        return {
            "success": True,
            "data": trending
        }
    except Exception as e:
        logger.error(f"Failed to get trending courses: {str(e)}")
        return {
            "success": False,
            "message": "Error fetching trending courses"
        }
