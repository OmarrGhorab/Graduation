import httpx
from app.config import settings
import logging
import time
from typing import Any, Dict, List, Optional

logger = logging.getLogger(__name__)

class CourseClient:
    def __init__(self):
        self.base_url = settings.COURSES_SERVICE_URL
        self.secret = settings.INTERNAL_SERVICE_SECRET
        self._catalog_cache: List[Dict[str, Any]] = []
        self._catalog_cache_fingerprint: Optional[str] = None
        self._catalog_cache_at: float = 0.0
        self._user_analytics_cache: Dict[str, Dict[str, Any]] = {}
        self._user_analytics_cache_at: Dict[str, float] = {}

    def get_cached_courses(self) -> List[Dict[str, Any]]:
        return list(self._catalog_cache)

    def set_cached_courses(self, courses: List[Dict[str, Any]], fingerprint: Optional[str] = None) -> None:
        self._catalog_cache = list(courses or [])
        self._catalog_cache_fingerprint = fingerprint
        self._catalog_cache_at = time.monotonic()

    def clear_cached_courses(self) -> None:
        self._catalog_cache = []
        self._catalog_cache_fingerprint = None
        self._catalog_cache_at = 0.0

    def get_catalog_fingerprint(self) -> Optional[str]:
        return self._catalog_cache_fingerprint

    def _is_catalog_cache_fresh(self) -> bool:
        if not self._catalog_cache:
            return False
        age = time.monotonic() - self._catalog_cache_at
        return age < settings.CATALOG_CACHE_TTL

    def get_cached_user_analytics_profile(self, user_id: str) -> Dict[str, Any]:
        cached = self._user_analytics_cache.get(user_id)
        if cached is None:
            return {}
        cached_at = self._user_analytics_cache_at.get(user_id, 0.0)
        age = time.monotonic() - cached_at
        if age >= settings.USER_PROFILE_CACHE_TTL:
            # Expired — remove from cache so next fetch is live
            self._user_analytics_cache.pop(user_id, None)
            self._user_analytics_cache_at.pop(user_id, None)
            return {}
        return dict(cached)

    def clear_cached_user_analytics_profile(self, user_id: str) -> None:
        self._user_analytics_cache.pop(str(user_id), None)
        self._user_analytics_cache_at.pop(str(user_id), None)

    async def get_all_courses(self):
        """Fetches the full course catalog from the internal endpoint."""
        if self._is_catalog_cache_fresh():
            return list(self._catalog_cache)
        async with httpx.AsyncClient() as client:
            try:
                response = await client.get(
                    f"{self.base_url}/api/v1/internal/courses",
                    headers={"x-internal-service-secret": self.secret},
                    timeout=10.0
                )
                response.raise_for_status()
                data = response.json()
                courses = data.get("data", [])
                self._catalog_cache = list(courses)
                self._catalog_cache_at = time.monotonic()
                logger.info(f"Course catalog refreshed: {len(courses)} courses")
                return courses
            except Exception as e:
                logger.error(f"Failed to fetch courses: {str(e)}")
                # Fall back to stale cache if available
                if self._catalog_cache:
                    logger.warning("Returning stale catalog cache after fetch failure")
                    return list(self._catalog_cache)
                return []

    async def get_user_analytics_profile(self, user_id: str):
        """Fetches the user's combined analytics profile from the internal endpoint."""
        cached = self._user_analytics_cache.get(user_id)
        if cached is not None:
            cached_at = self._user_analytics_cache_at.get(user_id, 0.0)
            age = time.monotonic() - cached_at
            if age < settings.USER_PROFILE_CACHE_TTL:
                return dict(cached)

        async with httpx.AsyncClient() as client:
            try:
                response = await client.get(
                    f"{self.base_url}/api/v1/internal/analytics/user/{user_id}",
                    headers={"x-internal-service-secret": self.secret},
                    timeout=10.0
                )
                response.raise_for_status()
                data = response.json()
                payload = data.get("data", {})
                self._user_analytics_cache[user_id] = dict(payload)
                self._user_analytics_cache_at[user_id] = time.monotonic()
                return payload
            except Exception as e:
                logger.error(f"Failed to fetch user analytics: {str(e)}")
                return {}

    async def get_activity(self, user_id: str, period: str = "weekly"):
        """Fetches the user's activity (watch time, completed lessons) for reports based on period."""
        async with httpx.AsyncClient() as client:
            try:
                response = await client.get(
                    f"{self.base_url}/api/v1/internal/reports/student/{user_id}",
                    params={"period": period},
                    headers={"x-internal-service-secret": self.secret},
                    timeout=10.0
                )
                response.raise_for_status()
                data = response.json()
                return data.get("data", {})
            except Exception as e:
                logger.error(f"Failed to fetch {period} activity: {str(e)}")
                return {}

course_client = CourseClient()
