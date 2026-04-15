"""
Chat API Routes — SSE-streaming chatbot endpoints.

All routes are authenticated via the shared ``get_current_user`` dependency.
"""

from fastapi import APIRouter, Depends, HTTPException, Query, File, UploadFile, Form
from starlette.responses import StreamingResponse

from app.api.dependencies import get_current_user
from app.schemas.chat import CreateChatRequest, SendMessageRequest, UpdateChatRequest
from app.services.chat_engine import chat_engine
import logging

router = APIRouter()
logger = logging.getLogger(__name__)


# ------------------------------------------------------------------ #
#  Session management
# ------------------------------------------------------------------ #


@router.post("")
async def create_chat(
    body: CreateChatRequest = None,
    user=Depends(get_current_user),
):
    """
    Create a new chat session.

    Body (optional):
        { "title": "My Chat" }

    If no title is given, one will be auto-generated from the first message.
    """
    try:
        title = body.title if body else None
        session = await chat_engine.create_session(user["user_id"], title)
        return {"success": True, "data": session}
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.get("")
async def list_chats(user=Depends(get_current_user)):
    """List all active chat sessions for the current user."""
    sessions = await chat_engine.list_sessions(user["user_id"])
    return {"success": True, "data": sessions}


@router.delete("/{chat_id}")
async def delete_chat(chat_id: str, user=Depends(get_current_user)):
    """Soft-delete a chat session."""
    deleted = await chat_engine.delete_session(user["user_id"], chat_id)
    if not deleted:
        raise HTTPException(status_code=404, detail="Chat session not found")
    return {"success": True, "message": "Chat session deleted"}


@router.patch("/{chat_id}")
async def update_chat(
    chat_id: str,
    body: UpdateChatRequest,
    user=Depends(get_current_user),
):
    """Update a chat session title."""
    updated = await chat_engine.update_session(user["user_id"], chat_id, body.title)
    if not updated:
        raise HTTPException(status_code=404, detail="Chat session not found")
    return {"success": True, "message": "Chat title updated"}


# ------------------------------------------------------------------ #
#  Messaging (SSE streaming)
# ------------------------------------------------------------------ #


@router.post("/{chat_id}/messages")
async def send_message(
    chat_id: str,
    body: SendMessageRequest,
    user=Depends(get_current_user),
):
    """
    Send a message and receive the AI response as a **Server-Sent Event**
    stream.

    SSE event types:
    - ``chunk``      — Partial AI text  ``{ "content": "..." }``
    - ``correction`` — Sanitised full text (only sent if code was stripped)
    - ``done``       — Final metadata ``{ "userMessageId", "assistantMessageId", "chatId" }``
    - ``error``      — Error details  ``{ "type", "message" }``
    """
    logger.info(f"Received message request for chat {chat_id} from user {user.get('user_id')}")
    return StreamingResponse(
        chat_engine.stream_response(
            user["user_id"], chat_id, body.message, body.media.model_dump() if body.media else None
        ),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
            "X-Accel-Buffering": "no",
        },
    )


@router.post("/{chat_id}/messages/binary")
async def send_message_binary(
    chat_id: str,
    message: str = Form(...),
    file: UploadFile = File(None),
    user=Depends(get_current_user),
):
    """
    Multimodal endpoint that accepts 'multipart/form-data' (real file uploads).
    Useful for testing from Postman with real files.
    """
    media_data = None
    if file:
        import base64
        
        # Validate file type
        allowed_image_types = ["image/jpeg", "image/jpg", "image/png", "image/webp", "image/heic", "image/heif"]
        allowed_audio_types = ["audio/wav", "audio/mp3", "audio/aiff", "audio/aac", "audio/ogg", "audio/flac"]
        
        if file.content_type not in allowed_image_types + allowed_audio_types:
            raise HTTPException(
                status_code=400, 
                detail=f"Unsupported file type: {file.content_type}. Supported types: {', '.join(allowed_image_types + allowed_audio_types)}"
            )
        
        content = await file.read()
        
        # Validate file size (max 20MB for images, 10MB for audio)
        max_size = 20 * 1024 * 1024 if file.content_type in allowed_image_types else 10 * 1024 * 1024
        if len(content) > max_size:
            raise HTTPException(
                status_code=400,
                detail=f"File too large. Maximum size: {max_size // (1024*1024)}MB"
            )
        
        if len(content) == 0:
            raise HTTPException(status_code=400, detail="Empty file uploaded")
        
        # Encode to base64 for consistency with the JSON endpoint
        base64_data = base64.b64encode(content).decode('utf-8')
        media_data = {
            "mimeType": file.content_type,
            "data": base64_data
        }
        
        logger.info(f"Received file: {file.filename}, type: {file.content_type}, size: {len(content)} bytes")

    return StreamingResponse(
        chat_engine.stream_response(
            user["user_id"], chat_id, message, media_data
        ),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
            "X-Accel-Buffering": "no",
        },
    )


# ------------------------------------------------------------------ #
#  History retrieval
# ------------------------------------------------------------------ #


@router.get("/{chat_id}/messages")
async def get_history(
    chat_id: str,
    page: int = Query(1, ge=1),
    limit: int = Query(50, ge=1, le=100),
    user=Depends(get_current_user),
):
    """Get paginated message history for a chat session."""
    result = await chat_engine.get_chat_history(
        user["user_id"], chat_id, page, limit
    )
    if result is None:
        raise HTTPException(status_code=404, detail="Chat session not found")

    return {
        "success": True,
        "data": result["messages"],
        "total": result["total"],
        "page": result["page"],
        "limit": result["limit"],
    }
