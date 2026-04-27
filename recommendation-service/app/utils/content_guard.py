"""
Content Guard — Dual-layer filter to restrict the chatbot from generating code
or answering off-topic questions.

Layer 1 (Input):  Rejects messages that request code generation.
Layer 2 (Output): Sanitizes AI responses that accidentally contain code blocks.
"""

import re
from typing import Tuple

# ---------------------------------------------------------------------------
# INPUT GUARD — patterns that indicate a request to generate code
# ---------------------------------------------------------------------------

_CODE_REQUEST_PATTERNS = [
    # Direct code requests
    r"\b(write|create|generate|give\s+me|show\s+me|provide|make|build)\b.{0,30}\b(code|script|program|function|class|method|snippet|implementation)\b",
    r"\b(implement|code)\b.{0,20}\b(this|that|it|for\s+me)\b",
    # Language-specific requests
    r"\b(write|create|code|make|build)\b.{0,20}\b(in\s+)?(python|java|javascript|typescript|c\+\+|c#|go|rust|ruby|php|swift|kotlin|dart|sql|html|css|bash|shell|powershell)\b",
    # "Give me the code"
    r"\bgive\s+me\s+(a\s+)?(the\s+)?(source\s+)?code\b",
    # "Can you code ..."
    r"\bcan\s+you\s+(write|code|create|generate|make)\b",
    # "Solve this in code"
    r"\bsolve\b.{0,15}\b(in\s+code|programmatically|using\s+code)\b",
    # "Debug / fix this code" — still off-limits (we explain concepts, not debug)
    r"\b(debug|fix|correct|refactor)\b.{0,20}\b(code|script|program|function|bug)\b",
]

_compiled_input_patterns = [
    re.compile(p, re.IGNORECASE) for p in _CODE_REQUEST_PATTERNS
]

# ---------------------------------------------------------------------------
# OUTPUT GUARD — patterns that indicate the AI included code in its reply
# ---------------------------------------------------------------------------

_CODE_BLOCK_RE = re.compile(r"```[\s\S]*?```", re.MULTILINE)
_INLINE_CODE_LONG_RE = re.compile(r"`[^`]{20,}`")  # substantial inline code

_REFUSAL_TEXT = (
    "I'm here to help you understand educational concepts from your courses. "
    "I can't generate or provide code, but I'd be happy to explain the underlying "
    "concepts in plain language! Could you rephrase your question?"
)

_SANITIZED_PLACEHOLDER = (
    "\n\n_[Code example removed — I'm designed to explain concepts in words "
    "rather than provide code. Let me rephrase this explanation for you.]_\n\n"
)


def validate_input(message: str) -> Tuple[bool, str]:
    """
    Checks whether a user message is requesting code generation.

    Returns
    -------
    (is_valid, rejection_reason)
        ``is_valid`` is True when the message is acceptable.
        ``rejection_reason`` contains a friendly refusal if not.
    """
    for pattern in _compiled_input_patterns:
        if pattern.search(message):
            return False, _REFUSAL_TEXT

    return True, ""


def sanitize_output(response: str) -> str:
    """
    Removes markdown code blocks from an AI response if the model
    accidentally generates code despite the system prompt restrictions.

    Returns the cleaned string (or the original if no code was found).
    """
    cleaned = _CODE_BLOCK_RE.sub(_SANITIZED_PLACEHOLDER, response)
    return cleaned
