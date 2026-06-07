import logging
from typing import Optional

from fastapi import APIRouter, Depends, Query

from app.api.dependencies import get_current_user
from app.services.course_search_service import course_autocomplete, semantic_course_search

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
    suggestions = await course_autocomplete(user["user_id"], search, limit)
    return {
        "success": True,
        "data": suggestions,
        "meta": {
            "limit": min(max(limit, 1), 20),
            "search": search,
        },
    }
