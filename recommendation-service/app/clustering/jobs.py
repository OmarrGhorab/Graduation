import logging
from typing import Dict

from app.clustering.cluster_service import ClusterService
from app.models.database import SessionLocal
from app.services.course_client import course_client
from opentelemetry import trace
from time import perf_counter

logger = logging.getLogger(__name__)
tracer = trace.get_tracer(__name__)


async def run_clustering_job_for_users(user_ids: list[str]) -> Dict:
    db = SessionLocal()
    start = perf_counter()
    try:
        with tracer.start_as_current_span("recommendation.cluster.assign") as span:
            user_profiles: Dict[str, Dict] = {}
            for user_id in user_ids:
                profile = await course_client.get_user_analytics_profile(user_id)
                if profile:
                    user_profiles[user_id] = profile

            service = ClusterService(db)
            assignments = await service.cluster_users(user_profiles)
            duration_ms = (perf_counter() - start) * 1000
            span.set_attribute("cluster.requested_users", len(user_ids))
            span.set_attribute("cluster.processed_users", len(user_profiles))
            span.set_attribute("cluster.assigned_users", len(assignments))
            span.set_attribute("cluster.duration_ms", duration_ms)
            logger.info(
                "clustering_job_completed",
                extra={
                    "requested_users": len(user_ids),
                    "processed_users": len(user_profiles),
                    "assigned_users": len(assignments),
                    "duration_ms": round(duration_ms, 2),
                },
            )
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
