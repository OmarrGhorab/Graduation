import asyncio
import logging
from app.models.database import engine
from sqlalchemy import text

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

async def run_migration():
    logger.info("Connecting to database to add missing columns...")
    
    with engine.connect() as conn:
        try:
            conn.execute(text("ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS media_url VARCHAR(500);"))
            conn.execute(text("ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS media_type VARCHAR(50);"))
            conn.commit()
            logger.info("Successfully added 'media_url' and 'media_type' to 'chat_messages'.")
        except Exception as e:
            logger.error(f"Error executing migration: {e}")

if __name__ == "__main__":
    asyncio.run(run_migration())
