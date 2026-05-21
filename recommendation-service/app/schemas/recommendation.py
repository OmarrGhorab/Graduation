from pydantic import BaseModel
from typing import List, Optional, Dict, Any
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


class RecommendationV2Item(BaseModel):
    courseId: str
    title: Optional[str] = None
    score: int
    confidence: Optional[float] = None
    matchReason: Optional[str] = None
    reason: Optional[str] = None
    priority: str
    source: List[str] = []
    clusterContribution: Optional[float] = 0.0


class RecommendationExplainResponse(BaseModel):
    success: bool
    data: Dict[str, Any]
