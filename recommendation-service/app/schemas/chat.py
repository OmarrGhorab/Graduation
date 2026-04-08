from pydantic import BaseModel, Field
from typing import List, Optional
from datetime import datetime


class CreateChatRequest(BaseModel):
    title: Optional[str] = Field(None, max_length=200)


class MediaPart(BaseModel):
    mimeType: str
    data: str  # Base64 data

class SendMessageRequest(BaseModel):
    message: str = Field(..., min_length=1, max_length=2000)
    media: Optional[MediaPart] = None


class ChatMessageResponse(BaseModel):
    id: str
    role: str
    content: str
    createdAt: datetime

    class Config:
        from_attributes = True


class ChatSessionResponse(BaseModel):
    id: str
    title: Optional[str] = None
    createdAt: datetime
    updatedAt: datetime
    lastMessage: Optional[str] = None


class ChatSessionListResponse(BaseModel):
    success: bool
    data: List[ChatSessionResponse]


class ChatHistoryResponse(BaseModel):
    success: bool
    data: List[ChatMessageResponse]
    total: int
    page: int
    limit: int


class ResponseModel(BaseModel):
    success: bool
    data: Optional[dict] = None
    message: Optional[str] = None
