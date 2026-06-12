from typing import Any, Dict, List, Optional

from pydantic import BaseModel, Field


class ToolDefinition(BaseModel):
    name: str
    description: str
    input_model: type[BaseModel]
    output_model: type[BaseModel]


class ToolInvokeRequest(BaseModel):
    name: str = Field(..., min_length=1, max_length=128)
    arguments: Dict[str, Any] = Field(default_factory=dict)


class ToolInvokeResponse(BaseModel):
    name: str
    success: bool
    data: Dict[str, Any]
    error: Optional[str] = None


class GetUserProfileInput(BaseModel):
    user_id: str = Field(..., min_length=1, max_length=64)


class GetUserProfileOutput(BaseModel):
    user_profile: Dict[str, Any]


class GetUserHistoryInput(BaseModel):
    user_id: str = Field(..., min_length=1, max_length=64)


class GetUserHistoryOutput(BaseModel):
    enrolled_course_ids: List[str] = Field(default_factory=list)
    cart_subjects: List[str] = Field(default_factory=list)
    interests: List[str] = Field(default_factory=list)
    subject_preferences: List[Dict[str, Any]] = Field(default_factory=list)
    preview_interests: List[Dict[str, Any]] = Field(default_factory=list)
    behavior_query: str = ""


class GetRecentCartActivityInput(BaseModel):
    user_id: str = Field(..., min_length=1, max_length=64)


class GetRecentCartActivityOutput(BaseModel):
    cart_subjects: List[str] = Field(default_factory=list)


class SearchRelevantCoursesInput(BaseModel):
    user_id: str = Field(..., min_length=1, max_length=64)
    query: str = Field(..., min_length=1, max_length=1000)
    top_k: int = Field(default=20, ge=1, le=50)
    exclude_course_ids: List[str] = Field(default_factory=list)


class SearchRelevantCoursesOutput(BaseModel):
    courses: List[Dict[str, Any]] = Field(default_factory=list)


class GetTrendingCoursesInput(BaseModel):
    limit: int = Field(default=10, ge=1, le=50)


class GetTrendingCoursesOutput(BaseModel):
    courses: List[Dict[str, Any]] = Field(default_factory=list)


class GetUserClusterInput(BaseModel):
    user_id: str = Field(..., min_length=1, max_length=64)


class GetUserClusterOutput(BaseModel):
    cluster: Optional[Dict[str, Any]] = None


class GetSimilarUsersInput(BaseModel):
    cluster_id: int = Field(..., ge=0)
    limit: int = Field(default=20, ge=1, le=100)


class GetSimilarUsersOutput(BaseModel):
    users: List[Dict[str, Any]] = Field(default_factory=list)


class GetClusterTopCoursesInput(BaseModel):
    cluster_id: int = Field(..., ge=0)
    limit: int = Field(default=10, ge=1, le=50)


class GetClusterTopCoursesOutput(BaseModel):
    courses: List[Dict[str, Any]] = Field(default_factory=list)
