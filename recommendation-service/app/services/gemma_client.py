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

    async def generate_recommendations(self, prompt: str):
        """
        Sends a prompt to Gemma 4 and expects a JSON response.
        """
        try:
            # Using the simplified direct client call for Gemini/Gemma models
            # Specifically requesting JSON output
            response = self.client.models.generate_content(
                model=self.model_id,
                contents=prompt,
                config=types.GenerateContentConfig(
                    response_mime_type="application/json",
                ),
            )
            
            if not response.text:
                logger.error("Empty response from AI model")
                return []

            # Parse the JSON response
            recommendations = json.loads(response.text)
            return recommendations
        
        except Exception as e:
            logger.error(f"Error calling AI model: {str(e)}")
            # Fallback or empty list
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

gemma_client = GemmaClient()
