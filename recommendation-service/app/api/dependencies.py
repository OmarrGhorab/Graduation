from fastapi import Header, HTTPException, Depends
from app.config import settings
import httpx
import logging

logger = logging.getLogger(__name__)

async def get_current_user(authorization: str = Header(None)):
    """
    Validates the JWT token by calling the Auth Service.
    This mimics the behavior of the Go services.
    """
    if not authorization or not authorization.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="Invalid or missing authorization header")

    token = authorization.split(" ")[1]
    
    async with httpx.AsyncClient() as client:
        try:
            response = await client.post(
                f"{settings.AUTH_SERVICE_URL}/api/v1/internal/validate-token",
                json={"token": token},
                headers={"x-internal-service-secret": settings.INTERNAL_SERVICE_SECRET}
            )
            
            if response.status_code != 200:
                raise HTTPException(status_code=401, detail="Invalid or expired token")
            
            data = response.json()
            if not data.get("valid"):
                raise HTTPException(status_code=401, detail="Unauthorized")
                
            return {
                "user_id": data.get("userId"),
                "role": data.get("role")
            }
        except Exception as e:
            logger.error(f"Auth validation error: {str(e)}")
            raise HTTPException(status_code=401, detail="Authentication failed")
