import httpx
from app.config import settings
import logging

logger = logging.getLogger(__name__)

class CourseClient:
    def __init__(self):
        self.base_url = settings.COURSES_SERVICE_URL
        self.secret = settings.INTERNAL_SERVICE_SECRET

    async def get_all_courses(self):
        """Fetches the full course catalog from the internal endpoint."""
        async with httpx.AsyncClient() as client:
            try:
                response = await client.get(
                    f"{self.base_url}/api/v1/internal/courses",
                    headers={"x-internal-service-secret": self.secret},
                    timeout=10.0
                )
                response.raise_for_status()
                data = response.json()
                return data.get("data", [])
            except Exception as e:
                logger.error(f"Failed to fetch courses: {str(e)}")
                return []

    async def get_user_analytics_profile(self, user_id: str):
        """Fetches the user's combined analytics profile from the internal endpoint."""
        async with httpx.AsyncClient() as client:
            try:
                response = await client.get(
                    f"{self.base_url}/api/v1/internal/analytics/user/{user_id}",
                    headers={"x-internal-service-secret": self.secret},
                    timeout=10.0
                )
                response.raise_for_status()
                data = response.json()
                return data.get("data", {})
            except Exception as e:
                logger.error(f"Failed to fetch user analytics: {str(e)}")
                return {}

course_client = CourseClient()
