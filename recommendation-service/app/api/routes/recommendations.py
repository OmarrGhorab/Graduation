from fastapi import APIRouter, Depends, HTTPException, BackgroundTasks
from app.services.recommendation_engine import get_personalized_recommendations
from app.api.dependencies import get_current_user
from typing import Dict
import logging
from app.models.database import SessionLocal
from app.clustering.cluster_service import ClusterService
from app.utils.api_response import error_response, success_response

router = APIRouter()
logger = logging.getLogger(__name__)

@router.get("/")
@router.get("")
async def get_my_recommendations(user = Depends(get_current_user)):
    """
    Returns a list of AI-powered course recommendations for the authenticated user.
    """
    user_id = user["user_id"]
    try:
        recommendations = await get_personalized_recommendations(user_id)
        return success_response(data=recommendations)
    except Exception as e:
        logger.error(f"Failed to get recommendations for {user_id}: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail=error_response("RECOMMENDATION_GENERATION_FAILED", "Could not generate recommendations"),
        )

@router.get("/explain")
@router.get("/explain/")
async def get_recommendations_explain(user = Depends(get_current_user)):
    """
    Returns reasoning summary and tool trace for latest v2 recommendation generation.
    """
    from app.services.recommendation_engine import get_recommendation_explanation
    user_id = user["user_id"]
    try:
        explanation = await get_recommendation_explanation(user_id)
        return success_response(data=explanation)
    except Exception as e:
        logger.error(f"Failed to get explanation for {user_id}: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail=error_response("RECOMMENDATION_EXPLANATION_FAILED", "Could not fetch recommendation explanation"),
        )

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
    return success_response(message="Recommendations cache cleared and refresh started in background")

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
    return success_response(message=f"Cache invalidated and background refresh started for user {user_id}")

@router.post("/test")
async def chat_test(data: Dict):
    """
    Directly test the AI connectivity with a message.
    Body: {"message": "Hi!"}
    """
    from app.services.gemma_client import gemma_client
    message = data.get("message", "Hello!")
    response = await gemma_client.chat(message)
    return success_response(data={"response": response})

@router.get("/trending")
@router.get("/trending/")
async def get_trending_courses():
    """
    Returns globally trending courses based on popularity and authority.
    """
    from app.services.recommendation_engine import get_trending_recommendations
    try:
        trending = await get_trending_recommendations()
        return success_response(data=trending)
    except Exception as e:
        logger.error(f"Failed to get trending courses: {str(e)}")
        return error_response("TRENDING_RECOMMENDATIONS_FAILED", "Error fetching trending courses")


@router.get("/clusters/{user_id}")
async def get_user_cluster(user_id: str):
    db = SessionLocal()
    try:
        service = ClusterService(db)
        cluster = service.get_user_cluster(user_id)
        if not cluster:
            raise HTTPException(
                status_code=404,
                detail=error_response("CLUSTER_ASSIGNMENT_NOT_FOUND", "Cluster assignment not found for user"),
            )
        return success_response(
            data={
                "userId": cluster.user_id,
                "clusterId": cluster.cluster_id,
                "distanceToCentroid": cluster.distance_to_centroid,
                "assignedAt": cluster.assigned_at.isoformat() if cluster.assigned_at else None,
            },
        )
    finally:
        db.close()


@router.get("/clusters/{cluster_id}/top-courses")
async def get_cluster_top_courses(cluster_id: int):
    db = SessionLocal()
    try:
        service = ClusterService(db)
        top_courses = service.list_top_courses_for_cluster(cluster_id)
        return success_response(
            data={
                "clusterId": cluster_id,
                "topCourses": top_courses,
            },
        )
    finally:
        db.close()
