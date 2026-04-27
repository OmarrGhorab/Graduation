from sqlalchemy import Column, String, DateTime, Text, ForeignKey
from sqlalchemy.sql import func
from app.models.database import Base
import uuid

class StudentReport(Base):
    __tablename__ = "student_reports"

    id = Column(String, primary_key=True, default=lambda: str(uuid.uuid4()))
    student_id = Column(String, index=True, nullable=False)
    parent_id = Column(String, index=True, nullable=False)
    student_name = Column(String, nullable=False)
    report_text = Column(Text, nullable=False)
    period = Column(String, nullable=False) # 'weekly' or 'monthly'
    language = Column(String, default="en")
    created_at = Column(DateTime(timezone=True), server_default=func.now())
