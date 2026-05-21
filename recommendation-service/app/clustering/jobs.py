import logging
from typing import Dict

from app.clustering.cluster_service import ClusterService
from app.models.database import SessionLocal
from app.services.course_client import course_client

logger = logging.getLogger(__name__)


async def run_clustering_job_for_users(user_ids: list[str]) -> Dict:
    db = SessionLocal()
    try:
        user_profiles: Dict[str, Dict] = {}
        for user_id in user_ids:
            profile = await course_client.get_user_analytics_profile(user_id)
            if profile:
                user_profiles[user_id] = profile

        service = ClusterService(db)
        assignments = await service.cluster_users(user_profiles)
        logger.info(f"Clustering job completed for {len(assignments)} users")
        return {
            "success": True,
            "processed": len(user_profiles),
            "assigned": len(assignments),
        }
    except Exception as exc:
        logger.error(f"Clustering job failed: {exc}", exc_info=True)
        return {"success": False, "error": str(exc)}
    finally:
        db.close()
