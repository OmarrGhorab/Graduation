from google import genai
from google.genai import types
from app.config import settings
import logging
import json

logger = logging.getLogger(__name__)

class GemmaClient:
    def __init__(self):
        self.client = genai.Client(api_key=settings.AI_API_KEY)
        self.model_id = settings.AI_MODEL
        logger.info(f"GemmaClient initialized with model: {self.model_id}")

    async def generate_recommendations(self, prompt: str):
        """
        Sends a prompt to Gemma 4 and expects a JSON response.
        """
        try:
            logger.info("Engaging AI model for recommendations...")
            response = self.client.models.generate_content(
                model=self.model_id,
                contents=prompt,
                config=types.GenerateContentConfig(
                    response_mime_type="application/json",
                ),
            )
            
            if not response.text:
                logger.error("Empty response text from AI model")
                return []

            logger.info(f"AI response body: {response.text[:200]}...")

            # Parse the JSON response
            recommendations = json.loads(response.text)
            
            # Defensive check: if it's a dict, convert to list if possible
            if isinstance(recommendations, dict):
                # Check for common wrapper fields
                if "recommendations" in recommendations:
                    recommendations = recommendations["recommendations"]
                elif "data" in recommendations:
                    recommendations = recommendations["data"]
            
            if not isinstance(recommendations, list):
                logger.warning(f"AI returned non-list data: {type(recommendations)}")
                return []
                
            return recommendations
        
        except Exception as e:
            logger.error(f"Error calling AI model: {str(e)}", exc_info=True)
            return []

    async def chat(self, message: str):
        """
        Sends a simple chat message to the model for testing purposes.
        """
        try:
            response = self.client.models.generate_content(
                model=self.model_id,
                contents=message
            )
            return response.text
        except Exception as e:
            logger.error(f"Chat error: {str(e)}")
            return f"Error: {str(e)}"

    async def stream_chat(self, system_prompt: str, messages: list, media: dict = None):
        """
        Streams a multi-turn chat response via async generator.
        Supports multimodal inputs if 'media' is provided for the latest turn.

        Parameters
        ----------
        system_prompt : str
            The restrictive system instruction.
        messages : list[dict]
            Conversation history.
        media : dict, optional
            A dict with {"mimeType": "...", "data": "base64..."} for the latest message.
        """
        try:
            import base64
            logger.info(f"Starting stream_chat with {len(messages)} messages, media: {media is not None}")
            
            # Build the Content objects for multi-turn conversation
            contents = []
            
            # For the last message, if media exists, we attach it
            for i, msg in enumerate(messages):
                parts = [types.Part(text=msg["content"])]
                
                # If this is the last message AND media was provided
                if i == len(messages) - 1 and media:
                    try:
                        # Handle both base64 string and raw bytes
                        if isinstance(media["data"], str):
                            raw_data = base64.b64decode(media["data"])
                        elif isinstance(media["data"], bytes):
                            raw_data = media["data"]
                        else:
                            raise ValueError(f"Unsupported media data type: {type(media['data'])}")
                        
                        # Validate the data is not empty
                        if not raw_data or len(raw_data) == 0:
                            raise ValueError("Media data is empty")
                        
                        parts.append(
                            types.Part(
                                inline_data=types.Blob(
                                    data=raw_data,
                                    mime_type=media["mimeType"]
                                )
                            )
                        )
                        logger.info(f"Successfully attached media: {media['mimeType']}, size: {len(raw_data)} bytes")
                    except Exception as media_err:
                        logger.error(f"Failed to process media part: {media_err}")
                        raise ValueError(f"Invalid media format: {str(media_err)}")

                contents.append(
                    types.Content(
                        role=msg["role"],
                        parts=parts,
                    )
                )

            config = types.GenerateContentConfig(
                system_instruction=system_prompt,
            )

            logger.info(f"Calling model {self.model_id} with {len(contents)} content items")
            
            # Use the async streaming API
            async_response = self.client.aio.models.generate_content_stream(
                model=self.model_id,
                contents=contents,
                config=config,
            )

            logger.info("Streaming response started")
            chunk_count = 0
            async for chunk in async_response:
                if chunk.text:
                    chunk_count += 1
                    yield chunk.text
            
            logger.info(f"Streaming completed with {chunk_count} chunks")

        except Exception as e:
            logger.error(f"Streaming chat error: {str(e)}")
            raise

gemma_client = GemmaClient()
