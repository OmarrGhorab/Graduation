import asyncio
import logging
from typing import List, Optional
import httpx
from fpdf import FPDF
from app.config import settings
from app.services.course_client import course_client
from app.services.chat_engine import chat_engine
from app.services.gemma_client import gemma_client
from app.models.database import SessionLocal
from app.models.report import StudentReport

logger = logging.getLogger(__name__)

class ReportEngine:
    """Orchestrates AI-generated progress reports for parents."""

    async def generate_parent_report(
        self, student_id: str, student_name: str, language: str = "en", period: str = "weekly"
    ) -> Optional[str]:
        """
        1. Fetches activity for period (Go)
        2. Fetches chatbot questions (Python)
        3. Generates AI summary (Gemma) in the student/parent's language
        4. Returns the report text
        """
        try:
            # Step 1 & 2: Parallel data fetch
            activity, chat_topics = await asyncio.gather(
                course_client.get_activity(student_id, period),
                chat_engine.get_weekly_topics(student_id) # Chat topics stay weekly for simplicity or can be expanded
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
                "chatbot_queries": chat_topics[:10],
                "period": period
            }

            # Step 4: AI Summary in specified language
            report_text = await self._generate_ai_summary(prompt_data, language)
            return report_text

        except Exception as e:
            logger.error(f"Failed to generate parent report: {str(e)}", exc_info=True)
            return None

    async def _generate_ai_summary(self, data: dict, language: str) -> str:
        """Calls Gemma 4 to create a friendly parent summary."""
        
        period_text = "weekly" if data["period"] == "weekly" else "monthly"
        lang_instruction = f"The entire report must be written in {language}."
        if language == "ar":
            lang_instruction += " Use formal Arabic (Fusha) but keep it friendly."

        system_prompt = (
            "You are an AI Education Assistant for a learning platform. "
            f"Your task is to write a {period_text} progress report for a parent about their child's learning. "
            "Tone: Encouraging, professional, and clear. Avoid technical jargon. "
            "Style: Human-readable, 2-3 short paragraphs. "
            f"{lang_instruction} "
            "Focus: Celebrate achievements, mention specific subjects, and highlight what they were curious about."
        )

        user_prompt = (
            f"Child Name: {data['student_name']}\n"
            f"Activity over the last {period_text}:\n"
            f"- Total Video Learning Time: {data['watch_time_minutes']} minutes\n"
            f"- Lessons Fully Completed: {data['lessons_completed']}\n"
            f"- Most Active Subjects: {', '.join(data['top_subjects']) if data['top_subjects'] else 'General'}\n"
            f"- Topics they asked the AI Chatbot about: {', '.join(data['chatbot_queries']) if data['chatbot_queries'] else 'None yet'}\n\n"
            "Please write the report now."
        )

        response = ""
        async for chunk in gemma_client.stream_chat(system_prompt, [{"role": "user", "content": user_prompt}]):
            print(chunk, end="", flush=True)
            response += chunk
        
        print("\n") # New line after streaming
        logger.info(f"Full Report Generated (first 100 chars): {response[:100]}...")
        return response

    def generate_pdf(self, student_name: str, report_text: str, language: str = "en", period: str = "weekly") -> bytes:
        """Generates a nicely formatted PDF of the report."""
        try:
            logger.info("PDF: Starting generation...")
            pdf = FPDF()
            pdf.add_page()
            
            # Helper to clean text for FPDF (Helvetica only supports Latin-1)
            def clean_text(text):
                if not text: return ""
                # Replace common unicode characters that crash Helvetica
                replacements = {
                    "’": "'", "‘": "'", "“": '"', "”": '"', 
                    "–": "-", "—": "-", "…": "...", "•": "*",
                    "é": "e", "á": "a", "í": "i", "ó": "o", "ú": "u",
                    "ü": "u", "ñ": "n"
                }
                for old, new in replacements.items():
                    text = text.replace(old, new)
                
                # Final pass: Force to latin-1 and ignore what doesn't fit
                return text.encode('latin-1', 'replace').decode('latin-1')

            safe_report = clean_text(report_text)
            safe_name = clean_text(student_name)
            
            # Header
            logger.info("PDF: Writing Header...")
            pdf.set_font("Helvetica", "B", 20)
            pdf.set_text_color(40, 50, 110)
            
            period_label = "Weekly" if period == "weekly" else "Monthly"
            title = f"{period_label} Learning Report"
            if language == "ar":
                title = f"{period_label} Learning Report (Arabic Summary)"

            pdf.cell(0, 20, title, ln=True, align="C")
            
            # Student Info
            logger.info("PDF: Writing Student Info...")
            pdf.set_font("Helvetica", "B", 14)
            pdf.set_text_color(0, 0, 0)
            pdf.cell(0, 10, f"Student: {safe_name}", ln=True)
            pdf.set_font("Helvetica", "", 10)
            from datetime import datetime
            pdf.cell(0, 10, f"Date: {datetime.now().strftime('%Y-%m-%d')}", ln=True)
            pdf.ln(10)
            
            # Report Body
            logger.info("PDF: Writing Body...")
            pdf.set_font("Helvetica", "", 12)
            pdf.multi_cell(0, 10, safe_report)
            
            # Footer
            logger.info("PDF: Writing Footer...")
            pdf.set_y(-30)
            pdf.set_font("Helvetica", "I", 8)
            pdf.set_text_color(128, 128, 128)
            pdf.cell(0, 10, "Powered by Gemma 4 AI Learning Platform", align="C")
            
            logger.info("PDF: Generation Complete.")
            pdf_data = bytes(pdf.output())
            logger.info(f"PDF: Size is {len(pdf_data) / 1024:.2f} KB")
            return pdf_data
        except Exception as e:
            logger.error(f"PDF Generation failed: {str(e)}", exc_info=True)
            raise e

    async def run_report_background(
        self, parent_id: str, student_id: str, student_name: str, language: str = "en", period: str = "weekly"
    ):
        """Full background flow: Generate -> Save to DB -> Notify."""
        report_text = await self.generate_parent_report(student_id, student_name, language, period)
        if report_text:
            # 1. Save to Database for history
            try:
                db = SessionLocal()
                new_report = StudentReport(
                    student_id=student_id,
                    parent_id=parent_id,
                    student_name=student_name,
                    report_text=report_text,
                    period=period,
                    language=language
                )
                db.add(new_report)
                db.commit()
                db.close()
                logger.info(f"Report saved to DB for student {student_id}")
            except Exception as e:
                logger.error(f"Failed to save report to DB: {str(e)}")

            # 2. Notify Parent
            await self.send_notification(parent_id, student_id, report_text)
        else:
            logger.error(f"Background report failed for student {student_id}")

    async def send_notification(self, parent_id: str, student_id: str, content: str):
        """Calls the Notification Service to send the report."""
        url = f"{settings.NOTIFICATION_SERVICE_URL}/api/v1/notifications/publish"
        headers = {"x-internal-service-secret": settings.INTERNAL_SERVICE_SECRET}
        
        payload = {
            "userId": parent_id,
            "type": "parent_report_ready",
            "studentId": student_id,
            "summary": content[:500] + ("..." if len(content) > 500 else ""),
            "fullReport": content,
            "title": "New Progress Report Ready",
            "body": f"Your child's progress report is ready. View it now!"
        }

        try:
            async with httpx.AsyncClient() as client:
                response = await client.post(url, json=payload, headers=headers)
                response.raise_for_status()
                logger.info(f"Notification sent to parent {parent_id}")
        except Exception as e:
            logger.error(f"Failed to send notification: {str(e)}")

report_engine = ReportEngine()
