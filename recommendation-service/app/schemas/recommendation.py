from pydantic import BaseModel
from typing import List, Optional
from uuid import UUID

class RecommendationItem(BaseModel):
    courseId: str
    score: int
    matchReason: str
    priority: str

class RecommendationResponse(BaseModel):
    success: bool
    data: List[RecommendationItem]
    
class RefreshResponse(BaseModel):
    success: bool
    message: str
