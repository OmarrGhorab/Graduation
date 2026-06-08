import uuid
from datetime import datetime

from sqlalchemy import Column, DateTime, Float, Integer, JSON, String

from app.models.database import Base


class UserCluster(Base):
    __tablename__ = "user_clusters"

    id = Column(String, primary_key=True, default=lambda: str(uuid.uuid4()))
    user_id = Column(String, unique=True, index=True, nullable=False)
    cluster_id = Column(Integer, index=True, nullable=False)
    embedding_version = Column(String, nullable=False, default="bge-small-en-v1.5")
    feature_version = Column(String, nullable=False, default="v1")
    distance_to_centroid = Column(Float, nullable=True)
    assigned_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    metadata_json = Column(JSON, nullable=True)


class ClusterMetadata(Base):
    __tablename__ = "cluster_metadata"

    id = Column(String, primary_key=True, default=lambda: str(uuid.uuid4()))
    cluster_id = Column(Integer, unique=True, index=True, nullable=False)
    label = Column(String, nullable=False, default="general")
    centroid_vector = Column(JSON, nullable=True)
    top_subjects = Column(JSON, nullable=True)
    top_courses = Column(JSON, nullable=True)
    user_count = Column(Integer, nullable=False, default=0)
    model_version = Column(String, nullable=False, default="kmeans-v1")
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow, nullable=False)
