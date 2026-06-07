import logging
from typing import Dict, List

from app.clustering.cluster_service import ClusterService
from app.jobs.embedding_jobs import refresh_user_embeddings
from app.models.database import SessionLocal
from app.models.recommendation import RecommendationHistory
from app.services.course_client import course_client
from opentelemetry import trace
from time import perf_counter

logger = logging.getLogger(__name__)
tracer = trace.get_tracer(__name__)


def gather_known_user_ids() -> List[str]:
    """Collect distinct user IDs to seed the clustering job.

    There is no "list all users" upstream endpoint, so we derive the cohort
    from two sources in the shared database:
      1. users who have learning analytics (the meaningful cohort to cluster)
      2. users who have previously received recommendations
    The union ensures a user is clusterable as soon as they have any activity.
    """
    from sqlalchemy import text

    db = SessionLocal()
    user_ids: set[str] = set()
    try:
        rows = db.query(RecommendationHistory.user_id).distinct().all()
        user_ids.update(str(r[0]) for r in rows if r[0])
    except Exception as exc:
        logger.warning(f"Failed to gather users from recommendation history: {exc}")
    try:
        rows = db.execute(text("SELECT DISTINCT user_id FROM public.user_course_analytics")).fetchall()
        user_ids.update(str(r[0]) for r in rows if r[0])
    except Exception as exc:
        logger.warning(f"Failed to gather users from course analytics: {exc}")
    finally:
        db.close()
    return list(user_ids)


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
                    try:
                        await refresh_user_embeddings(user_id, profile)
                    except Exception as exc:
                        logger.warning(f"User vector refresh failed for {user_id}: {exc}")

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
