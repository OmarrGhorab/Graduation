import logging
from collections import defaultdict
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

# Dimension of the text embedding (BAAI/bge-small-en-v1.5); the clusters Qdrant
# collection is created with this size, while KMeans feature vectors append extra
# numeric features on top.
EMBEDDING_DIMENSION = 384


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
        # Standardise features so large-magnitude numeric features (e.g. total
        # watch time) don't dominate the normalised text-embedding dimensions.
        if matrix.shape[0] > 1:
            std = matrix.std(axis=0)
            std[std == 0] = 1.0
            matrix = (matrix - matrix.mean(axis=0)) / std
        # Keep clusters meaningfully populated: target ~4 users per cluster so a
        # collaborative signal can emerge, never more clusters than users.
        cluster_count = max(1, min(settings.CLUSTER_COUNT, round(len(user_ids) / 4))) if len(user_ids) > 1 else 1
        model = KMeans(n_clusters=cluster_count, random_state=42, n_init=10)
        labels = model.fit_predict(matrix)

        # Aggregate per-cluster course/subject preferences from member behaviour
        cluster_prefs = self._aggregate_cluster_preferences(user_ids, labels, user_profiles)

        assignments: List[Dict] = []
        for idx, user_id in enumerate(user_ids):
            cluster_id = int(labels[idx])
            centroid = model.cluster_centers_[cluster_id].tolist()
            vector = feature_rows[idx]
            # Distance is measured in the (scaled) space the model was fit on.
            distance = float(np.linalg.norm(matrix[idx] - model.cluster_centers_[cluster_id]))
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

        # Persist cluster-level artefacts ONCE per distinct cluster (not per user,
        # which would attempt duplicate inserts for the same cluster_id).
        for cluster_id in sorted(set(int(l) for l in labels)):
            centroid = model.cluster_centers_[cluster_id].tolist()
            # Qdrant cluster vectors are an optional artefact; the boost signal is
            # served from ClusterMetadata in Postgres. The feature vector mixes a
            # 384-d text embedding with numeric features, so only the embedding
            # slice is stored (matching the collection dim). Never let a vector
            # store hiccup abort the DB persistence below.
            try:
                await vector_store.upsert_cluster_vector(
                    str(cluster_id),
                    centroid[:EMBEDDING_DIMENSION],
                    {
                        "cluster_id": cluster_id,
                        "user_count": int((labels == cluster_id).sum()),
                        "model_version": "kmeans-v1",
                    },
                )
            except Exception as exc:
                logger.warning(f"Cluster vector upsert skipped for cluster {cluster_id}: {exc}")
            self._upsert_cluster_metadata(cluster_id, centroid, labels, cluster_prefs.get(cluster_id, {}))

        self.db.commit()
        return assignments

    def _aggregate_cluster_preferences(
        self,
        user_ids: List[str],
        labels: "np.ndarray",
        user_profiles: Dict[str, Dict],
    ) -> Dict[int, Dict]:
        """For each cluster, rank the courses and subjects its members engage with.

        This is the collaborative-filtering signal: courses popular among similar
        users (the cluster) are surfaced so they can boost candidates for other
        members who have not enrolled in them yet.
        """
        per_cluster_courses: Dict[int, Dict[str, float]] = defaultdict(lambda: defaultdict(float))
        per_cluster_course_members: Dict[int, Dict[str, set]] = defaultdict(lambda: defaultdict(set))
        per_cluster_subjects: Dict[int, Dict[str, float]] = defaultdict(lambda: defaultdict(float))

        for idx, user_id in enumerate(user_ids):
            cluster_id = int(labels[idx])
            profile = user_profiles.get(user_id, {}) or {}
            for item in profile.get("AllAnalytics", []) or []:
                if not isinstance(item, dict):
                    continue
                course_id = item.get("CourseID")
                completion = float(item.get("CompletionPct", 0) or 0)
                engagement = float(item.get("EngagementScore", 0) or 0)
                # Weight an interaction by how strongly the user engaged with it
                weight = 1.0 + (completion / 100.0) + min(engagement / 100.0, 1.0)
                if course_id:
                    cid = str(course_id)
                    per_cluster_courses[cluster_id][cid] += weight
                    per_cluster_course_members[cluster_id][cid].add(user_id)
                subject_name = item.get("SubjectName")
                if subject_name:
                    per_cluster_subjects[cluster_id][str(subject_name)] += weight

        prefs: Dict[int, Dict] = {}
        for cluster_id in set(int(l) for l in labels):
            course_weights = per_cluster_courses.get(cluster_id, {})
            max_course_weight = max(course_weights.values(), default=0.0) or 1.0
            top_courses = sorted(
                (
                    {
                        "courseId": cid,
                        "score": round(weight / max_course_weight, 4),
                        "memberCount": len(per_cluster_course_members[cluster_id].get(cid, set())),
                        "weight": round(weight, 4),
                    }
                    for cid, weight in course_weights.items()
                ),
                key=lambda c: (c["memberCount"], c["weight"]),
                reverse=True,
            )[:20]

            subject_weights = per_cluster_subjects.get(cluster_id, {})
            max_subject_weight = max(subject_weights.values(), default=0.0) or 1.0
            top_subjects = sorted(
                (
                    {"subject": name, "score": round(weight / max_subject_weight, 4)}
                    for name, weight in subject_weights.items()
                ),
                key=lambda s: s["score"],
                reverse=True,
            )[:10]

            prefs[cluster_id] = {"top_courses": top_courses, "top_subjects": top_subjects}
        return prefs

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

    def _upsert_cluster_metadata(
        self,
        cluster_id: int,
        centroid: List[float],
        labels: np.ndarray,
        prefs: Optional[Dict] = None,
    ) -> None:
        prefs = prefs or {}
        top_subjects = prefs.get("top_subjects", [])
        top_courses = prefs.get("top_courses", [])
        existing = self.db.query(ClusterMetadata).filter(ClusterMetadata.cluster_id == cluster_id).first()
        user_count = int((labels == cluster_id).sum())
        if existing:
            existing.centroid_vector = centroid
            existing.user_count = user_count
            existing.model_version = "kmeans-v1"
            existing.top_subjects = top_subjects
            existing.top_courses = top_courses
            existing.updated_at = datetime.utcnow()
        else:
            row = ClusterMetadata(
                cluster_id=cluster_id,
                label=f"cluster-{cluster_id}",
                centroid_vector=centroid,
                top_subjects=top_subjects,
                top_courses=top_courses,
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

    def get_course_affinity_for_user(self, user_id: str) -> Dict[str, float]:
        """Return a {courseId: affinity 0..1} map from the user's cluster.

        Used by hybrid retrieval to boost candidate courses that are popular
        among behaviourally similar users (collaborative filtering signal).
        Returns an empty map when the user has no cluster assignment yet.
        """
        cluster = self.get_user_cluster(user_id)
        if not cluster:
            return {}
        meta = self.get_cluster_metadata(cluster.cluster_id)
        if not meta or not meta.top_courses:
            return {}
        affinity: Dict[str, float] = {}
        for entry in meta.top_courses:
            if not isinstance(entry, dict):
                continue
            cid = entry.get("courseId")
            score = entry.get("score")
            if cid is not None and score is not None:
                affinity[str(cid)] = float(score)
        return affinity
