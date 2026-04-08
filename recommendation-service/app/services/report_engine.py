import asyncio
import logging
from typing import List, Optional
import httpx
from fpdf import FPDF
from app.config import settings
from app.services.course_client import course_client
from app.services.chat_engine import chat_engine
from app.services.gemma_client import gemma_client

logger = logging.getLogger(__name__)

class ReportEngine:
    """Orchestrates AI-generated progress reports for parents."""

    async def generate_parent_report(
        self, student_id: str, student_name: str, language: str = "en"
    ) -> Optional[str]:
        """
        1. Fetches weekly activity (Go)
        2. Fetches chatbot questions (Python)
        3. Generates AI summary (Gemma) in the student/parent's language
        4. Returns the report text
        """
        try:
            # Step 1 & 2: Parallel data fetch
            activity, chat_topics = await asyncio.gather(
                course_client.get_weekly_activity(student_id),
                chat_engine.get_weekly_topics(student_id)
            )

            # Step 3: Format data for Gemma
            watch_time_mins = activity.get("avgSessionDuration", 0) // 60
            lessons_completed = int(activity.get("avgCompletionPct", 0))
            top_subjects = [s.get("subjectName") for s in activity.get("subjectPreferences", [])[:3]]
            
            prompt_data = {
                "student_name": student_name,
                "watch_time_minutes": watch_time_mins,
                "lessons_completed": lessons_completed,
                "top_subjects": top_subjects,
                "chatbot_queries": chat_topics[:10]
            }

            # Step 4: AI Summary in specified language
            report_text = await self._generate_ai_summary(prompt_data, language)
            return report_text

        except Exception as e:
            logger.error(f"Failed to generate parent report: {str(e)}", exc_info=True)
            return None

    async def _generate_ai_summary(self, data: dict, language: str) -> str:
        """Calls Gemma 4 to create a friendly parent summary."""
        
        lang_instruction = f"The entire report must be written in {language}."
        if language == "ar":
            lang_instruction += " Use formal Arabic (Fusha) but keep it friendly."

        system_prompt = (
            "You are an AI Education Assistant for a learning platform. "
            "Your task is to write a weekly progress report for a parent about their child's learning. "
            "Tone: Encouraging, professional, and clear. Avoid technical jargon. "
            "Style: Human-readable, 2-3 short paragraphs. "
            f"{lang_instruction} "
            "Focus: Celebrate achievements, mention specific subjects, and highlight what they were curious about."
        )

        user_prompt = (
            f"Child Name: {data['student_name']}\n"
            f"Activity this week:\n"
            f"- Total Video Learning Time: {data['watch_time_minutes']} minutes\n"
            f"- Lessons Fully Completed: {data['lessons_completed']}\n"
            f"- Most Active Subjects: {', '.join(data['top_subjects']) if data['top_subjects'] else 'General'}\n"
            f"- Topics they asked the AI Chatbot about: {', '.join(data['chatbot_queries']) if data['chatbot_queries'] else 'None yet'}\n\n"
            "Please write the report now."
        )

        response = ""
        async for chunk in gemma_client.stream_chat(system_prompt, [{"role": "user", "content": user_prompt}]):
            response += chunk
        
        return response

    def generate_pdf(self, student_name: str, report_text: str, language: str = "en") -> bytes:
        """Generates a nicely formatted PDF of the report."""
        pdf = FPDF()
        pdf.add_page()
        
        # Header
        pdf.set_font("Helvetica", "B", 20)
        pdf.set_text_color(40, 50, 110)
        title = "Weekly Learning Report" if language != "ar" else "تقرير التعلم الأسبوعي"
        pdf.cell(0, 20, title, ln=True, align="C")
        
        # Student Info
        pdf.set_font("Helvetica", "B", 14)
        pdf.set_text_color(0, 0, 0)
        pdf.cell(0, 10, f"Student: {student_name}", ln=True)
        pdf.set_font("Helvetica", "", 10)
        from datetime import datetime
        pdf.cell(0, 10, f"Date: {datetime.now().strftime('%Y-%m-%d')}", ln=True)
        pdf.ln(10)
        
        # Report Body
        pdf.set_font("Helvetica", "", 12)
        # Handle line breaks from AI
        pdf.multi_cell(0, 10, report_text)
        
        # Footer
        pdf.set_y(-30)
        pdf.set_font("Helvetica", "I", 8)
        pdf.set_text_color(128, 128, 128)
        pdf.cell(0, 10, "Powered by Gemma 4 AI Learning Platform", align="C")
        
        return pdf.output()

    async def send_notification(self, parent_id: str, student_id: str, content: str):
        """Calls the Notification Service to send the report."""
        logger.info(f"Sending Parent Report to {parent_id} for student {student_id}...")

report_engine = ReportEngine()
