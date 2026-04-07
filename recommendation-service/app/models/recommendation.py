from sqlalchemy import Column, String, Float, DateTime, Boolean, JSON
from sqlalchemy.dialects.postgresql import UUID
from app.models.database import Base
import uuid
from datetime import datetime

class RecommendationHistory(Base):
    __tablename__ = "recommendation_history"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    user_id = Column(UUID(as_uuid=True), index=True)
    course_id = Column(UUID(as_uuid=True))
    score = Column(Float)
    match_reason = Column(String)
    match_type = Column(String)  # "subject_match", "interest_match", etc.
    was_clicked = Column(Boolean, default=False)
    was_enrolled = Column(Boolean, default=False)
    model_version = Column(String)
    created_at = Column(DateTime, default=datetime.utcnow)
