import logging
from datetime import datetime
from typing import Dict, List, Optional

import numpy as np
from sklearn.cluster import KMeans
from sqlalchemy.orm import Session

from app.clustering.feature_builder import build_user_behavior_summary, build_user_numeric_features
from app.config import settings
from app.models.cluster import ClusterMetadata, UserCluster
from app.retrieval.embedding_service import embedding_service
from app.retrieval.vector_store import vector_store

logger = logging.getLogger(__name__)


class ClusterService:
    def __init__(self, db: Session):
        self.db = db

    async def _build_feature_vector(self, user_id: str, user_profile: Dict) -> List[float]:
        summary = build_user_behavior_summary(user_id, user_profile)
        text_vector = await embedding_service.embed_text(summary, normalize=True)
        numeric = build_user_numeric_features(user_profile)
        numeric_vector = [
            numeric["courses_count"],
            numeric["watch_time_total"],
            numeric["completion_avg"],
            numeric["engagement_avg"],
            numeric["cart_subjects_count"],
            numeric["top_category_count"],
        ]
        return text_vector + numeric_vector

    async def cluster_users(self, user_profiles: Dict[str, Dict]) -> List[Dict]:
        if not user_profiles:
            return []

        feature_rows: List[List[float]] = []
        user_ids: List[str] = []
        for user_id, profile in user_profiles.items():
            user_ids.append(user_id)
            feature_rows.append(await self._build_feature_vector(user_id, profile))

        matrix = np.array(feature_rows, dtype=float)
        cluster_count = min(settings.CLUSTER_COUNT, len(user_ids)) or 1
        model = KMeans(n_clusters=cluster_count, random_state=42, n_init=10)
        labels = model.fit_predict(matrix)

        assignments: List[Dict] = []
        for idx, user_id in enumerate(user_ids):
            cluster_id = int(labels[idx])
            centroid = model.cluster_centers_[cluster_id].tolist()
            vector = feature_rows[idx]
            distance = float(np.linalg.norm(np.array(vector) - np.array(model.cluster_centers_[cluster_id])))
            assignment = {
                "user_id": user_id,
                "cluster_id": cluster_id,
                "distance_to_centroid": distance,
                "feature_vector": vector,
                "metadata": {
                    "assigned_at": datetime.utcnow().isoformat(),
                    "feature_version": "v1",
                },
            }
            assignments.append(assignment)
            self._upsert_user_cluster(assignment)
            await vector_store.upsert_cluster_vector(
                str(cluster_id),
                centroid,
                {
                    "cluster_id": cluster_id,
                    "user_count": int((labels == cluster_id).sum()),
                    "model_version": "kmeans-v1",
                },
            )
            self._upsert_cluster_metadata(cluster_id, centroid, labels)

        self.db.commit()
        return assignments

    def _upsert_user_cluster(self, assignment: Dict) -> None:
        existing = self.db.query(UserCluster).filter(UserCluster.user_id == assignment["user_id"]).first()
        if existing:
            existing.cluster_id = assignment["cluster_id"]
            existing.distance_to_centroid = assignment["distance_to_centroid"]
            existing.metadata_json = assignment["metadata"]
            existing.assigned_at = datetime.utcnow()
        else:
            row = UserCluster(
                user_id=assignment["user_id"],
                cluster_id=assignment["cluster_id"],
                distance_to_centroid=assignment["distance_to_centroid"],
                metadata_json=assignment["metadata"],
            )
            self.db.add(row)
        # commit is handled once per clustering job

    def _upsert_cluster_metadata(self, cluster_id: int, centroid: List[float], labels: np.ndarray) -> None:
        existing = self.db.query(ClusterMetadata).filter(ClusterMetadata.cluster_id == cluster_id).first()
        user_count = int((labels == cluster_id).sum())
        if existing:
            existing.centroid_vector = centroid
            existing.user_count = user_count
            existing.model_version = "kmeans-v1"
            existing.top_subjects = existing.top_subjects or []
            existing.top_courses = existing.top_courses or []
            existing.updated_at = datetime.utcnow()
        else:
            row = ClusterMetadata(
                cluster_id=cluster_id,
                label=f"cluster-{cluster_id}",
                centroid_vector=centroid,
                top_subjects=[],
                top_courses=[],
                user_count=user_count,
                model_version="kmeans-v1",
            )
            self.db.add(row)
        # commit is handled once per clustering job

    def get_user_cluster(self, user_id: str) -> Optional[UserCluster]:
        return self.db.query(UserCluster).filter(UserCluster.user_id == user_id).first()

    def get_cluster_metadata(self, cluster_id: int) -> Optional[ClusterMetadata]:
        return self.db.query(ClusterMetadata).filter(ClusterMetadata.cluster_id == cluster_id).first()

    def list_top_courses_for_cluster(self, cluster_id: int) -> List[Dict]:
        meta = self.get_cluster_metadata(cluster_id)
        if not meta or not meta.top_courses:
            return []
        return meta.top_courses
