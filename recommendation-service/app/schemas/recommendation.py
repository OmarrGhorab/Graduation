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
    error: Optional[Dict[str, Any]] = None
    message: Optional[str] = None
    
class RefreshResponse(BaseModel):
    success: bool
    data: Optional[Dict[str, Any]] = None
    error: Optional[Dict[str, Any]] = None
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
    error: Optional[Dict[str, Any]] = None
    message: Optional[str] = None
