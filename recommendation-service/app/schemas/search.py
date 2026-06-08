from typing import Literal

from pydantic import BaseModel, Field


class SearchFeedbackRequest(BaseModel):
    query: str = Field(..., min_length=1, max_length=200)
    courseId: str = Field(..., min_length=1, max_length=100)
    eventType: Literal["click", "preview", "watch", "enroll"]


class SearchAnalyticsQueryItem(BaseModel):
    query: str
    totalSearches: int
    zeroResultSearches: int
    zeroResultRate: float
    totalClicks: int
    totalPreviews: int
    totalWatches: int
    totalEnrolls: int
    lastSearchedAt: str | None
    lastFeedbackAt: str | None


class SearchAnalyticsCourseItem(BaseModel):
    query: str
    courseId: str
    totalScore: float
    totalEvents: int
    clickCount: int
    previewCount: int
    watchCount: int
    enrollCount: int
    lastEventAt: str | None
