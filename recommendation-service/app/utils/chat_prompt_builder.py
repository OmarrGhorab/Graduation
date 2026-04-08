"""
Chat Prompt Builder — Constructs the system prompt and conversation history
for the restricted education-only chatbot.
"""

from typing import List, Dict


def build_chat_system_prompt(courses: List[Dict]) -> str:
    """
    Builds the system prompt that restricts the chatbot to education content
    and provides course context to ground its answers.
    """
    # Build course catalog for context
    course_context = ""
    for course in courses:
        subject = course.get("subject", {}).get("name", "General")
        title = course.get("title", "Untitled")
        description = (course.get("description", "") or "")[:120]
        course_context += f"- {title} (Subject: {subject}): {description}\n"

    system_prompt = f"""You are an AI teaching assistant for an online education platform.

━━━━ STRICT RULES — FOLLOW AT ALL TIMES ━━━━

1. SCOPE: You MUST ONLY answer questions related to educational content from the courses listed below.

2. NO CODE: You MUST NEVER generate, write, or provide code in ANY programming language. This includes:
   - Code snippets, scripts, functions, classes, methods
   - SQL queries, shell commands, configuration files
   - Pseudocode that resembles real code
   - Markdown code blocks (```)
   If asked for code, politely refuse and explain the concept in plain words instead.

3. ON-TOPIC ONLY: You MUST NEVER answer questions unrelated to the platform's educational content.
   If a user asks about something outside scope, respond:
   "I can only help with topics covered in our platform's courses. Feel free to ask me about any of the available subjects!"

4. TEACHING STYLE:
   - Use clear, concise, and pedagogically sound explanations
   - Use analogies, real-world examples, and step-by-step breakdowns
   - You may reference specific courses when relevant to guide the student's learning path
   - Encourage curiosity and deeper exploration of topics

5. FORMATTING:
   - Use bullet points and numbered lists for structured explanations
   - Use **bold** for key terms
   - Do NOT use code blocks or inline code formatting

━━━━ AVAILABLE COURSES ON THE PLATFORM ━━━━

{course_context if course_context else "No courses currently available."}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Remember: Your role is to TEACH and EXPLAIN concepts. Never write code. Never go off-topic."""

    return system_prompt


def build_conversation_messages(history: List[Dict], current_message: str) -> list:
    """
    Converts internal message history into the format expected by the
    Gemini/Gemma API (roles: "user" and "model").
    """
    messages = []

    for msg in history:
        # Map our "assistant" role to Gemini's "model" role
        role = "user" if msg["role"] == "user" else "model"
        messages.append({"role": role, "content": msg["content"]})

    # Append the new user message
    messages.append({"role": "user", "content": current_message})

    return messages


def generate_chat_title(first_message: str) -> str:
    """
    Auto-generates a short, clean chat title from the user's first message.
    """
    title = first_message.strip()

    # Remove question marks for cleaner titles
    title = title.rstrip("?").strip()

    if len(title) > 60:
        # Truncate at a word boundary
        truncated = title[:57]
        last_space = truncated.rfind(" ")
        if last_space > 25:
            title = truncated[:last_space] + "..."
        else:
            title = truncated + "..."

    return title or "New Chat"
