import logging
from typing import Optional

from fastapi import APIRouter, Depends, Query

from app.api.dependencies import get_current_user
from app.schemas.search import SearchFeedbackRequest
from app.services.course_search_service import (
    course_autocomplete,
    get_top_clicked_queries,
    get_top_query_course_pairs,
    get_zero_result_queries,
    log_recent_search,
    record_search_feedback,
    semantic_course_search,
)

router = APIRouter()
logger = logging.getLogger(__name__)


@router.get("/")
@router.get("")
async def search_courses(
    search: str = Query(..., min_length=1),
    page: int = Query(1, ge=1),
    limit: int = Query(10, ge=1, le=100),
    subjectId: Optional[str] = None,
    subjectName: Optional[str] = None,
    teacherId: Optional[str] = None,
    deliveryType: Optional[str] = None,
    isPaid: Optional[bool] = None,
    billingType: Optional[str] = None,
    status: Optional[str] = None,
    minPrice: Optional[float] = Query(None, ge=0),
    maxPrice: Optional[float] = Query(None, ge=0),
    user=Depends(get_current_user),
):
    filters = {
        "subjectId": subjectId,
        "subjectName": subjectName,
        "teacherId": teacherId,
        "deliveryType": deliveryType,
        "isPaid": isPaid,
        "billingType": billingType,
        "status": status,
        "minPrice": minPrice,
        "maxPrice": maxPrice,
    }
    await log_recent_search(user["user_id"], search)
    payload = await semantic_course_search(
        user_id=user["user_id"],
        search=search,
        filters=filters,
        page=page,
        limit=limit,
    )
    return {
        "success": True,
        "data": payload["data"],
        "meta": payload["meta"],
    }


@router.get("/autocomplete")
@router.get("/autocomplete/")
async def autocomplete_courses(
    search: str = Query(..., min_length=1),
    limit: int = Query(8, ge=1, le=20),
    user=Depends(get_current_user),
):
    await log_recent_search(user["user_id"], search)
    suggestions = await course_autocomplete(user["user_id"], search, limit)
    return {
        "success": True,
        "data": suggestions,
        "meta": {
            "limit": min(max(limit, 1), 20),
            "search": search,
        },
    }


@router.post("/feedback")
@router.post("/feedback/")
async def record_feedback(
    body: SearchFeedbackRequest,
    user=Depends(get_current_user),
):
    await record_search_feedback(body.query, body.courseId, body.eventType)
    return {
        "success": True,
        "data": {
            "recorded": True,
            "query": body.query,
            "courseId": body.courseId,
            "eventType": body.eventType,
        },
    }


@router.get("/analytics/top-clicked")
@router.get("/analytics/top-clicked/")
async def top_clicked_queries(
    limit: int = Query(10, ge=1, le=100),
    user=Depends(get_current_user),
):
    return {
        "success": True,
        "data": get_top_clicked_queries(limit),
        "meta": {"limit": limit},
    }


@router.get("/analytics/zero-results")
@router.get("/analytics/zero-results/")
async def zero_result_queries(
    limit: int = Query(10, ge=1, le=100),
    user=Depends(get_current_user),
):
    return {
        "success": True,
        "data": get_zero_result_queries(limit),
        "meta": {"limit": limit},
    }


@router.get("/analytics/top-query-courses")
@router.get("/analytics/top-query-courses/")
async def top_query_courses(
    limit: int = Query(10, ge=1, le=100),
    user=Depends(get_current_user),
):
    return {
        "success": True,
        "data": get_top_query_course_pairs(limit),
        "meta": {"limit": limit},
    }
