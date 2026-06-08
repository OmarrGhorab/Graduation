import uuid
from datetime import datetime

from sqlalchemy import Column, DateTime, Float, Integer, String

from app.models.database import Base


class SearchFeedbackEvent(Base):
    __tablename__ = "search_feedback_events"

    id = Column(String, primary_key=True, default=lambda: str(uuid.uuid4()))
    user_id = Column(String, index=True, nullable=True)
    query = Column(String, nullable=False)
    normalized_query = Column(String, index=True, nullable=False)
    course_id = Column(String, index=True, nullable=False)
    event_type = Column(String, nullable=False)
    weight = Column(Float, nullable=False, default=0.0)
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)


class SearchFeedbackAggregate(Base):
    __tablename__ = "search_feedback_aggregates"

    id = Column(String, primary_key=True, default=lambda: str(uuid.uuid4()))
    normalized_query = Column(String, index=True, nullable=False)
    course_id = Column(String, index=True, nullable=False)
    total_score = Column(Float, nullable=False, default=0.0)
    total_events = Column(Integer, nullable=False, default=0)
    click_count = Column(Integer, nullable=False, default=0)
    preview_count = Column(Integer, nullable=False, default=0)
    watch_count = Column(Integer, nullable=False, default=0)
    enroll_count = Column(Integer, nullable=False, default=0)
    last_event_at = Column(DateTime, default=datetime.utcnow, nullable=False)


class SearchQueryAnalytics(Base):
    __tablename__ = "search_query_analytics"

    id = Column(String, primary_key=True, default=lambda: str(uuid.uuid4()))
    normalized_query = Column(String, unique=True, index=True, nullable=False)
    display_query = Column(String, nullable=False)
    total_searches = Column(Integer, nullable=False, default=0)
    zero_result_searches = Column(Integer, nullable=False, default=0)
    total_clicks = Column(Integer, nullable=False, default=0)
    total_previews = Column(Integer, nullable=False, default=0)
    total_watches = Column(Integer, nullable=False, default=0)
    total_enrolls = Column(Integer, nullable=False, default=0)
    last_searched_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    last_feedback_at = Column(DateTime, nullable=True)
