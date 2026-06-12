from app.config import settings
import asyncio
import base64
import json
import logging
import re

import httpx
from typing import AsyncGenerator

logger = logging.getLogger(__name__)

_RETRYABLE_CODES = {429, 500, 502, 503, 504}
_MAX_RETRIES = 4
_BASE_DELAY = 2.0
_JSON_BLOCK_RE = re.compile(r"```(?:json)?\s*([\s\S]*?)```", re.IGNORECASE)


async def _with_retry(fn, *args, max_retries: int = _MAX_RETRIES, base_delay: float = _BASE_DELAY, **kwargs):
    for attempt in range(max_retries):
        try:
            return await fn(*args, **kwargs)
        except httpx.HTTPStatusError as exc:
            status_code = exc.response.status_code
            if status_code not in _RETRYABLE_CODES or attempt == max_retries - 1:
                raise
            delay = base_delay * (2 ** attempt)
            logger.warning(f"AI API {status_code} on attempt {attempt + 1}, retrying in {delay:.0f}s...")
            await asyncio.sleep(delay)
        except (httpx.ConnectError, httpx.ReadTimeout) as exc:
            if attempt == max_retries - 1:
                raise
            delay = base_delay * (2 ** attempt)
            logger.warning(f"AI API connection error on attempt {attempt + 1}: {exc}. Retrying in {delay:.0f}s...")
            await asyncio.sleep(delay)
    raise RuntimeError("unreachable")


def _extract_json(text: str):
    """Extract the first JSON object or array from model output."""
    match = _JSON_BLOCK_RE.search(text)
    if match:
        candidate = match.group(1).strip()
        try:
            return json.loads(candidate)
        except json.JSONDecodeError:
            pass

    for start_char, end_char in [("{", "}"), ("[", "]")]:
        start = text.find(start_char)
        if start == -1:
            continue
        end = text.rfind(end_char)
        if end == -1 or end <= start:
            continue
        try:
            return json.loads(text[start:end + 1])
        except json.JSONDecodeError:
            continue

    return None


class GemmaClient:
    def __init__(self):
        self.api_key = settings.AI_API_KEY
        self.model_id = settings.AI_MODEL
        self.base_url = settings.AI_BASE_URL.rstrip("/")
        self.wire_api = settings.AI_WIRE_API
        self.reasoning_effort = settings.AI_REASONING_EFFORT
        self.store_responses = not settings.DISABLE_RESPONSE_STORAGE
        logger.info(f"AI client initialized: model={self.model_id} wire={self.wire_api}")

    def _headers(self):
        return {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json",
        }

    async def _chat_completion(
        self,
        messages: list,
        *,
        response_format: dict | None = None,
        stream: bool = False,
        timeout_seconds: float = 90.0,
        reasoning_effort: str | None = None,
        max_retries: int = _MAX_RETRIES,
        base_delay: float = _BASE_DELAY,
    ):
        if stream:
            return self._stream_responses_completion(
                messages,
                response_format=response_format,
                timeout_seconds=timeout_seconds,
                reasoning_effort=reasoning_effort,
            )
        return await self._responses_completion(
            messages,
            response_format=response_format,
            timeout_seconds=timeout_seconds,
            reasoning_effort=reasoning_effort,
            max_retries=max_retries,
            base_delay=base_delay,
        )

    def _responses_url(self):
        if self.base_url.endswith("/v1"):
            return f"{self.base_url}/responses"
        return f"{self.base_url}/v1/responses"

    @staticmethod
    def _responses_input(messages: list):
        input_messages = []
        instructions = None
        for message in messages:
            role = message.get("role", "user")
            content = message.get("content", "")
            if role == "system":
                instructions = content if isinstance(content, str) else json.dumps(content)
                continue
            if isinstance(content, str):
                input_messages.append({
                    "role": role,
                    "content": [{"type": "input_text", "text": content}],
                })
            else:
                converted = []
                for part in content:
                    if part.get("type") == "text":
                        converted.append({"type": "input_text", "text": part.get("text", "")})
                    elif part.get("type") == "image_url":
                        converted.append({"type": "input_image", "image_url": part.get("image_url", {}).get("url", "")})
                input_messages.append({"role": role, "content": converted})
        return instructions, input_messages

    async def _responses_completion(
        self,
        messages: list,
        *,
        response_format: dict | None = None,
        timeout_seconds: float = 90.0,
        reasoning_effort: str | None = None,
        max_retries: int = _MAX_RETRIES,
        base_delay: float = _BASE_DELAY,
    ):
        instructions, input_messages = self._responses_input(messages)
        payload = {
            "model": self.model_id,
            "input": input_messages,
            "store": self.store_responses,
        }
        if instructions:
            payload["instructions"] = instructions
        selected_effort = reasoning_effort if reasoning_effort is not None else self.reasoning_effort
        if selected_effort:
            payload["reasoning"] = {"effort": selected_effort}

        async def _request():
            async with httpx.AsyncClient(timeout=timeout_seconds) as client:
                response = await client.post(
                    self._responses_url(),
                    headers=self._headers(),
                    json=payload,
                )
                response.raise_for_status()
                return response.json()

        return await _with_retry(_request, max_retries=max_retries, base_delay=base_delay)

    async def _stream_responses_completion(
        self,
        messages: list,
        *,
        response_format: dict | None = None,
        timeout_seconds: float = 90.0,
        reasoning_effort: str | None = None,
    ) -> AsyncGenerator[str, None]:
        instructions, input_messages = self._responses_input(messages)
        payload = {
            "model": self.model_id,
            "input": input_messages,
            "store": self.store_responses,
            "stream": True,
        }
        if instructions:
            payload["instructions"] = instructions
        if response_format:
            payload["text"] = {"format": response_format}
        selected_effort = reasoning_effort if reasoning_effort is not None else self.reasoning_effort
        if selected_effort:
            payload["reasoning"] = {"effort": selected_effort}

        async with httpx.AsyncClient(timeout=timeout_seconds) as client:
            async with client.stream(
                "POST",
                self._responses_url(),
                headers=self._headers(),
                json=payload,
            ) as response:
                response.raise_for_status()

                current_event = None
                data_lines: list[str] = []

                async for raw_line in response.aiter_lines():
                    line = raw_line.strip()
                    if not line:
                        chunk = self._extract_stream_chunk(current_event, data_lines)
                        if chunk:
                            yield chunk
                        current_event = None
                        data_lines = []
                        continue

                    if line.startswith("event:"):
                        current_event = line[6:].strip()
                        continue

                    if line.startswith("data:"):
                        data_lines.append(line[5:].strip())

                chunk = self._extract_stream_chunk(current_event, data_lines)
                if chunk:
                    yield chunk

    @classmethod
    def _extract_stream_chunk(cls, event_name: str | None, data_lines: list[str]) -> str:
        if not data_lines:
            return ""

        raw_payload = "\n".join(data_lines).strip()
        if not raw_payload or raw_payload == "[DONE]":
            return ""

        try:
            payload = json.loads(raw_payload)
        except json.JSONDecodeError:
            return ""

        if event_name == "response.output_text.delta":
            delta = payload.get("delta")
            return delta if isinstance(delta, str) else ""

        if event_name == "response.refusal.delta":
            delta = payload.get("delta")
            return delta if isinstance(delta, str) else ""

        if event_name in {"response.output_text.done", "response.completed"}:
            text = cls._text_from_response(payload)
            return text if isinstance(text, str) else ""

        delta = payload.get("delta")
        if isinstance(delta, str):
            return delta

        text = payload.get("text")
        if isinstance(text, str):
            return text

        return ""

    @staticmethod
    def _text_from_response(response: dict) -> str:
        output_text = response.get("output_text")
        if isinstance(output_text, str):
            return output_text
        if isinstance(response.get("output"), list):
            chunks = []
            for item in response["output"]:
                for part in item.get("content", []):
                    if isinstance(part, dict) and part.get("type") in {"output_text", "text"}:
                        chunks.append(part.get("text", ""))
            if chunks:
                return "".join(chunks)
        try:
            content = response["choices"][0]["message"]["content"]
        except (KeyError, IndexError, TypeError):
            return ""
        if isinstance(content, str):
            return content
        if isinstance(content, list):
            return "".join(part.get("text", "") for part in content if isinstance(part, dict))
        return ""

    async def generate_recommendations(self, prompt: str):
        try:
            logger.info("Engaging AI model for recommendations...")
            response = await self._chat_completion([
                {
                    "role": "system",
                    "content": "Respond with a valid JSON array only. No markdown, no explanation.",
                },
                {"role": "user", "content": prompt},
            ])

            text = self._text_from_response(response)
            if not text:
                logger.error("Empty response text from AI model")
                return []

            logger.info(f"AI response body: {text[:200]}...")
            recommendations = _extract_json(text)
            if recommendations is None:
                logger.warning("Could not extract JSON from AI response")
                return []

            if isinstance(recommendations, dict):
                if "recommendations" in recommendations:
                    recommendations = recommendations["recommendations"]
                elif "data" in recommendations:
                    recommendations = recommendations["data"]

            if not isinstance(recommendations, list):
                logger.warning(f"AI returned non-list data: {type(recommendations)}")
                return []

            return recommendations
        except Exception as exc:
            logger.error(f"Error calling AI model: {exc}", exc_info=True)
            return []

    async def chat(self, message: str):
        try:
            response = await self._chat_completion([{"role": "user", "content": message}])
            return self._text_from_response(response)
        except Exception as exc:
            logger.error(f"Chat error: {exc}")
            return f"Error: {exc}"

    async def _generate_json(
        self,
        system_prompt: str,
        payload: dict,
        *,
        timeout_seconds: float = 90.0,
        reasoning_effort: str | None = None,
        max_retries: int = _MAX_RETRIES,
        base_delay: float = _BASE_DELAY,
    ):
        try:
            response = await self._chat_completion(
                [
                    {
                        "role": "system",
                        "content": system_prompt + "\n\nRespond with valid JSON only. No markdown, no explanation.",
                    },
                    {"role": "user", "content": json.dumps(payload)},
                ],
                response_format={"type": "json_object"},
                timeout_seconds=timeout_seconds,
                reasoning_effort=reasoning_effort,
                max_retries=max_retries,
                base_delay=base_delay,
            )
            text = self._text_from_response(response)
            if not text:
                return {}
            parsed = _extract_json(text)
            if parsed is None:
                logger.warning(f"Could not extract JSON from model output: {text[:300]}")
                return {}
            return parsed
        except Exception as exc:
            logger.error(f"Structured generation error: {exc}", exc_info=True)
            return {}

    async def plan_next_tool(self, system_prompt: str, payload: dict):
        result = await self._generate_json(system_prompt, payload)
        if not isinstance(result, dict):
            return {"done": True, "tool_name": None, "arguments": {}, "reasoning_summary": "Planner fallback"}
        return {
            "done": bool(result.get("done", False)),
            "tool_name": result.get("tool_name"),
            "arguments": result.get("arguments") or {},
            "reasoning_summary": result.get("reasoning_summary", ""),
        }

    async def rank_recommendation_candidates(self, system_prompt: str, payload: dict):
        result = await self._generate_json(
            system_prompt,
            payload,
            timeout_seconds=20.0,
            reasoning_effort="low",
            max_retries=2,
            base_delay=1.0,
        )
        if isinstance(result, list):
            return result
        if isinstance(result, dict):
            if "recommendations" in result and isinstance(result["recommendations"], list):
                return result["recommendations"]
            if "data" in result and isinstance(result["data"], list):
                return result["data"]
        return []

    @staticmethod
    def _message_with_media(message: dict, media: dict | None):
        if not media:
            return {"role": message["role"], "content": message["content"]}

        raw_data = media["data"]
        if isinstance(raw_data, bytes):
            encoded = base64.b64encode(raw_data).decode("utf-8")
        elif isinstance(raw_data, str):
            encoded = raw_data
        else:
            raise ValueError(f"Unsupported media data type: {type(raw_data)}")

        if not encoded:
            raise ValueError("Media data is empty")

        mime_type = media["mimeType"]
        return {
            "role": message["role"],
            "content": [
                {"type": "text", "text": message["content"]},
                {"type": "image_url", "image_url": {"url": f"data:{mime_type};base64,{encoded}"}},
            ],
        }

    async def stream_chat(self, system_prompt: str, messages: list, media: dict = None):
        request_messages = [{"role": "system", "content": system_prompt}] + [
            self._message_with_media(message, media if index == len(messages) - 1 else None)
            for index, message in enumerate(messages)
        ]

        streamed_any = False
        streamed_text = ""

        try:
            stream = await self._chat_completion(request_messages, stream=True)
            async for chunk in stream:
                if chunk:
                    streamed_any = True
                    streamed_text += chunk
                    yield chunk
            if streamed_any:
                return
        except Exception as exc:
            logger.warning("Falling back to buffered AI response streaming: %s", exc)

        response = await self._chat_completion(request_messages)
        text = self._text_from_response(response)
        if not text:
            return

        if streamed_text and text.startswith(streamed_text):
            remaining = text[len(streamed_text):]
        else:
            remaining = text

        for chunk in self._chunk_text_for_stream(remaining):
            yield chunk
            await asyncio.sleep(0.02)

    @staticmethod
    def _chunk_text_for_stream(text: str, target_size: int = 48) -> list[str]:
        if not text:
            return []

        normalized = text.strip()
        if not normalized:
            return []

        chunks: list[str] = []
        start = 0
        length = len(normalized)

        while start < length:
            end = min(start + target_size, length)
            if end < length:
                split_candidates = [
                    normalized.rfind(marker, start, end)
                    for marker in (" ", ".", ",", "!", "?", "\n")
                ]
                split_at = max(split_candidates)
                if split_at > start:
                    end = split_at + 1
            chunk = normalized[start:end]
            if chunk:
                chunks.append(chunk)
            start = end

        return chunks


gemma_client = GemmaClient()
