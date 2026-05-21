from typing import Any, Dict, List

from app.models.cluster import UserCluster
from app.models.database import SessionLocal
from app.clustering.cluster_service import ClusterService


async def get_user_cluster_tool(user_id: str) -> Dict[str, Any]:
    db = SessionLocal()
    try:
        service = ClusterService(db)
        cluster = service.get_user_cluster(user_id)
        if not cluster:
            return {"cluster": None}
        return {
            "cluster": {
                "user_id": cluster.user_id,
                "cluster_id": cluster.cluster_id,
                "distance_to_centroid": cluster.distance_to_centroid,
                "assigned_at": cluster.assigned_at.isoformat() if cluster.assigned_at else None,
            }
        }
    finally:
        db.close()


async def get_similar_users_tool(cluster_id: int, limit: int = 20) -> Dict[str, Any]:
    db = SessionLocal()
    try:
        rows = (
            db.query(UserCluster)
            .filter(UserCluster.cluster_id == cluster_id)
            .order_by(UserCluster.distance_to_centroid.asc())
            .limit(limit)
            .all()
        )
        users = [
            {
                "user_id": row.user_id,
                "cluster_id": row.cluster_id,
                "distance_to_centroid": row.distance_to_centroid,
            }
            for row in rows
        ]
        return {"users": users}
    finally:
        db.close()


async def get_cluster_top_courses_tool(cluster_id: int, limit: int = 10) -> Dict[str, Any]:
    db = SessionLocal()
    try:
        service = ClusterService(db)
        courses = service.list_top_courses_for_cluster(cluster_id)
        return {"courses": (courses or [])[:limit]}
    finally:
        db.close()
