"""
Chat Engine — Orchestrates the full chatbot flow:
session management, content filtering, prompt building,
Gemma 4 streaming, and message persistence.
"""

import asyncio
import json
import logging
from datetime import datetime
from typing import AsyncGenerator, List, Optional
from uuid import uuid4

import redis.asyncio as redis
from sqlalchemy import desc, func

from app.config import settings
from app.models.chat import ChatMessage, ChatSession
from app.models.database import SessionLocal
from app.services.course_client import course_client
from app.services.gemma_client import gemma_client
from app.utils.chat_prompt_builder import (
    build_chat_system_prompt,
    build_conversation_messages,
    generate_chat_title,
)
from app.utils.content_guard import sanitize_output, validate_input
import cloudinary
import cloudinary.uploader
from cloudinary.utils import cloudinary_url

logger = logging.getLogger(__name__)

# Configure Cloudinary
cloudinary.config(
    cloud_name=settings.CLOUDINARY_CLOUD_NAME,
    api_key=settings.CLOUDINARY_API_KEY,
    api_secret=settings.CLOUDINARY_API_SECRET,
    secure=True
)

# Reuse the recommendation service's Redis connection
redis_conn = redis.from_url(settings.REDIS_URL, decode_responses=True)


class ChatEngine:
    """Handles the complete chatbot lifecycle."""

    # ------------------------------------------------------------------ #
    #  Session CRUD
    # ------------------------------------------------------------------ #

    async def create_session(
        self, user_id: str, title: Optional[str] = None
    ) -> dict:
        """Creates a new chat session (enforces per-user limit)."""

        def _db_work():
            db = SessionLocal()
            try:
                active_count = (
                    db.query(func.count(ChatSession.id))
                    .filter(
                        ChatSession.user_id == user_id,
                        ChatSession.is_active == True,
                    )
                    .scalar()
                )

                if active_count >= settings.CHATBOT_MAX_ACTIVE_CHATS:
                    raise ValueError(
                        f"Maximum number of active chats "
                        f"({settings.CHATBOT_MAX_ACTIVE_CHATS}) reached. "
                        f"Please delete an existing chat first."
                    )

                session = ChatSession(
                    id=uuid4(),
                    user_id=user_id,
                    title=title,
                    is_active=True,
                )
                db.add(session)
                db.commit()
                db.refresh(session)

                return {
                    "id": str(session.id),
                    "title": session.title,
                    "createdAt": session.created_at.isoformat(),
                    "updatedAt": session.updated_at.isoformat(),
                }
            finally:
                db.close()

        return await asyncio.to_thread(_db_work)

    async def list_sessions(self, user_id: str) -> List[dict]:
        """Returns all active chat sessions for a user."""

        def _db_work():
            db = SessionLocal()
            try:
                sessions = (
                    db.query(ChatSession)
                    .filter(
                        ChatSession.user_id == user_id,
                        ChatSession.is_active == True,
                    )
                    .order_by(desc(ChatSession.updated_at))
                    .all()
                )

                result = []
                for s in sessions:
                    last_msg = (
                        db.query(ChatMessage)
                        .filter(ChatMessage.chat_session_id == s.id)
                        .order_by(desc(ChatMessage.created_at))
                        .first()
                    )
                    result.append(
                        {
                            "id": str(s.id),
                            "title": s.title or "New Chat",
                            "createdAt": s.created_at.isoformat(),
                            "updatedAt": s.updated_at.isoformat(),
                            "lastMessage": (
                                last_msg.content[:100] if last_msg else None
                            ),
                        }
                    )
                return result
            finally:
                db.close()

        return await asyncio.to_thread(_db_work)

    async def delete_session(self, user_id: str, chat_id: str) -> bool:
        """Soft-deletes a chat session."""

        def _db_work():
            db = SessionLocal()
            try:
                session = (
                    db.query(ChatSession)
                    .filter(
                        ChatSession.id == chat_id,
                        ChatSession.user_id == user_id,
                        ChatSession.is_active == True,
                    )
                    .first()
                )
                if not session:
                    return False

                session.is_active = False
                db.commit()
                return True
            finally:
                db.close()

        return await asyncio.to_thread(_db_work)

    async def get_chat_history(
        self, user_id: str, chat_id: str, page: int = 1, limit: int = 50
    ) -> Optional[dict]:
        """Paginated chat history for a single session."""

        def _db_work():
            db = SessionLocal()
            try:
                # Verify ownership
                session = (
                    db.query(ChatSession)
                    .filter(
                        ChatSession.id == chat_id,
                        ChatSession.user_id == user_id,
                        ChatSession.is_active == True,
                    )
                    .first()
                )
                if not session:
                    return None

                total = (
                    db.query(func.count(ChatMessage.id))
                    .filter(ChatMessage.chat_session_id == chat_id)
                    .scalar()
                )

                offset = (page - 1) * limit
                messages = (
                    db.query(ChatMessage)
                    .filter(ChatMessage.chat_session_id == chat_id)
                    .order_by(ChatMessage.created_at)
                    .offset(offset)
                    .limit(limit)
                    .all()
                )

                return {
                    "messages": [
                        {
                            "id": str(m.id),
                            "role": m.role,
                            "content": m.content,
                            "createdAt": m.created_at.isoformat(),
                        }
                        for m in messages
                    ],
                    "total": total,
                    "page": page,
                    "limit": limit,
                }
            finally:
                db.close()

        return await asyncio.to_thread(_db_work)

    async def get_weekly_topics(self, user_id: str) -> List[str]:
        """Returns a list of unique chatbot query topics for a user in the last 7 days."""

        def _db_work():
            db = SessionLocal()
            try:
                seven_days_ago = datetime.utcnow().replace(hour=0, minute=0, second=0, microsecond=0)
                # Filter for last 7 days
                messages = (
                    db.query(ChatMessage)
                    .join(ChatSession)
                    .filter(
                        ChatSession.user_id == user_id,
                        ChatMessage.role == "user",
                        ChatMessage.created_at >= seven_days_ago
                    )
                    .all()
                )
                
                # Simple extraction: first 50 chars of each message (Gemma will summarize later)
                return [m.content[:100] for m in messages]
            finally:
                db.close()

        return await asyncio.to_thread(_db_work)

    # ------------------------------------------------------------------ #
    #  Internal helpers
    # ------------------------------------------------------------------ #

    async def _get_course_context(self) -> List[dict]:
        """Fetches all platform courses (cached in Redis for speed)."""
        cache_key = "chatbot:course_context"

        try:
            cached = await redis_conn.get(cache_key)
            if cached:
                return json.loads(cached)
        except Exception as e:
            logger.warning(f"Redis read error (courses): {e}")

        courses = await course_client.get_all_courses()

        try:
            await redis_conn.setex(
                cache_key,
                settings.CHATBOT_COURSE_CONTEXT_TTL,
                json.dumps(courses),
            )
        except Exception as e:
            logger.warning(f"Redis write error (courses): {e}")

        return courses

    def _load_history(self, chat_id: str) -> List[dict]:
        """Loads the last N messages for the AI context window (sync)."""
        db = SessionLocal()
        try:
            messages = (
                db.query(ChatMessage)
                .filter(ChatMessage.chat_session_id == chat_id)
                .order_by(desc(ChatMessage.created_at))
                .limit(settings.CHATBOT_MAX_CONTEXT_MESSAGES)
                .all()
            )
            # Reverse to chronological order
            messages = list(reversed(messages))
            return [
                {"role": m.role, "content": m.content} for m in messages
            ]
        finally:
            db.close()

    def _save_messages(
        self,
        chat_id: str,
        user_message: str,
        assistant_message: str,
        is_first_message: bool,
        has_title: bool,
        media_url: str = None,
        media_type: str = None,
    ) -> tuple:
        """Persists both the user and assistant messages (sync)."""
        db = SessionLocal()
        try:
            user_msg = ChatMessage(
                id=uuid4(),
                chat_session_id=chat_id,
                role="user",
                content=user_message,
                media_url=media_url,
                media_type=media_type,
            )
            assistant_msg = ChatMessage(
                id=uuid4(),
                chat_session_id=chat_id,
                role="assistant",
                content=assistant_message,
            )
            db.add(user_msg)
            db.add(assistant_msg)

            # Auto-generate title on first message if none provided
            session = (
                db.query(ChatSession)
                .filter(ChatSession.id == chat_id)
                .first()
            )
            if session:
                if is_first_message and not has_title:
                    session.title = generate_chat_title(user_message)
                session.updated_at = datetime.utcnow()

            db.commit()
            return str(user_msg.id), str(assistant_msg.id)
        finally:
            db.close()

    # ------------------------------------------------------------------ #
    #  Streaming response — the main entry point for chatting
    # ------------------------------------------------------------------ #

    async def stream_response(
        self, user_id: str, chat_id: str, message: str, media: dict = None
    ) -> AsyncGenerator[str, None]:
        """
        Full multimodal chat pipeline: media is attached to the current user turn.
        """

        # ── 1. Input guard (Text only) ──────────────────────────────────
        is_valid, rejection = validate_input(message)
        if not is_valid:
            yield self._sse("error", {
                "type": "content_blocked",
                "message": rejection,
            })
            return

        # ── 2. Verify session ownership ─────────────────────────────────
        def _verify():
            db = SessionLocal()
            try:
                session = (
                    db.query(ChatSession)
                    .filter(
                        ChatSession.id == chat_id,
                        ChatSession.user_id == user_id,
                        ChatSession.is_active == True,
                    )
                    .first()
                )
                if not session:
                    return None

                msg_count = (
                    db.query(func.count(ChatMessage.id))
                    .filter(ChatMessage.chat_session_id == chat_id)
                    .scalar()
                )
                return {
                    "has_title": session.title is not None,
                    "is_first_message": msg_count == 0,
                }
            finally:
                db.close()

        session_info = await asyncio.to_thread(_verify)
        if session_info is None:
            yield self._sse("error", {
                "type": "not_found",
                "message": "Chat session not found",
            })
            return

        # ── 3. Load history + courses in parallel ───────────────────────
        try:
            history, courses = await asyncio.gather(
                asyncio.to_thread(self._load_history, chat_id),
                self._get_course_context(),
            )

            # ── 4. Build prompt ─────────────────────────────────────────
            system_prompt = build_chat_system_prompt(courses)
            # Add media context note if present
            if media:
                media_type = "image" if "image" in media.get("mimeType", "") else "audio/voice"
                message_with_media = f"[User provided {media_type}] {message}"
            else:
                message_with_media = message
            
            conversation = build_conversation_messages(history, message)

            # ── 5. Stream from Gemma 4 (Passing media) ──────────────────
            full_response = ""
            async for chunk in gemma_client.stream_chat(
                system_prompt, conversation, media=media
            ):
                full_response += chunk
                yield self._sse("chunk", {"content": chunk})

            # ── 6. Output guard ─────────────────────────────────────────
            sanitized = sanitize_output(full_response)
            if sanitized != full_response:
                yield self._sse("correction", {"content": sanitized})
                full_response = sanitized

            # ── 7. Persist messages ─────────────────────────────────────
            # Save message with media tag and URL if exists
            media_url = None
            media_type = None
            # ── 7. Persist messages ─────────────────────────────────────
            media_url = None
            media_type = None
            
            if media:
                try:
                    # media can be a dict (base64) or a bytes object
                    if isinstance(media.get("data"), str):
                        # Base64 path
                        upload_file = f"data:{media['mimeType']};base64,{media['data']}"
                    else:
                        # Raw bytes/file path (used by binary endpoint)
                        upload_file = media["data"]

                    upload_res = await asyncio.to_thread(
                        cloudinary.uploader.upload,
                        upload_file,
                        folder=settings.CLOUDINARY_FOLDER,
                        resource_type="auto"
                    )
                    media_url = upload_res.get("secure_url")
                    media_type = "image" if "image" in media["mimeType"] else "audio"
                except Exception as e:
                    logger.error(f"Cloudinary upload failed: {e}")

            user_msg_id, assistant_msg_id = await asyncio.to_thread(
                self._save_messages,
                chat_id,
                message_with_media,
                full_response,
                session_info["is_first_message"],
                session_info["has_title"],
                media_url=media_url,
                media_type=media_type,
            )

            # ── 8. Done ────────────────────────────────────────────────
            yield self._sse("done", {
                "userMessageId": user_msg_id,
                "assistantMessageId": assistant_msg_id,
                "chatId": chat_id,
            })

        except Exception as e:
            logger.error(f"Chat streaming error: {str(e)}", exc_info=True)
            yield self._sse("error", {
                "type": "server_error",
                "message": f"AI Engine Error: {str(e)}",
            })

    # ------------------------------------------------------------------ #
    #  SSE formatting helper
    # ------------------------------------------------------------------ #

    @staticmethod
    def _sse(event: str, data: dict) -> str:
        """Formats a single Server-Sent Event."""
        return f"event: {event}\ndata: {json.dumps(data)}\n\n"


# Singleton
chat_engine = ChatEngine()
