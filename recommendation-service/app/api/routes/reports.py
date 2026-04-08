from fastapi import APIRouter, Depends, HTTPException, Response
from app.api.dependencies import get_current_user
from app.schemas.chat import ResponseModel
from app.services.report_engine import report_engine
import logging

logger = logging.getLogger(__name__)

router = APIRouter()

@router.post("/parent/student/{student_id}/trigger", response_model=ResponseModel)
async def trigger_parent_report(
    student_id: str,
    student_name: str = "Omar", 
    language: str = "en",
    current_user: dict = Depends(get_current_user)
):
    """
    Manually triggers the generation of an AI progress report for a student.
    Supported languages: "en", "ar", "fr", etc.
    """
    if current_user["role"] != "PARENT":
        raise HTTPException(status_code=403, detail="Only parents can trigger progress reports.")

    report_text = await report_engine.generate_parent_report(student_id, student_name, language)
    
    if not report_text:
        raise HTTPException(status_code=500, detail="Failed to generate the AI report.")

    await report_engine.send_notification(current_user["user_id"], student_id, report_text)

    return {
        "success": True,
        "data": {
            "studentId": student_id,
            "studentName": student_name,
            "language": language,
            "reportSummary": report_text
        }
    }

@router.get("/parent/student/{student_id}/download")
async def download_parent_report(
    student_id: str,
    student_name: str = "Omar",
    language: str = "en",
    current_user: dict = Depends(get_current_user)
):
    """Generates and returns the progress report as a downloadable PDF."""
    if current_user["role"] != "PARENT":
         raise HTTPException(status_code=403, detail="Unauthorized")

    # 1. Generate the AI text first
    report_text = await report_engine.generate_parent_report(student_id, student_name, language)
    if not report_text:
        raise HTTPException(status_code=500, detail="Failed to generate report text.")

    # 2. Convert to PDF
    pdf_bytes = report_engine.generate_pdf(student_name, report_text, language)

    # 3. Return as a file response
    return Response(
        content=pdf_bytes,
        media_type="application/pdf",
        headers={
            "Content-Disposition": f"attachment; filename=progress_report_{student_id}.pdf"
        }
    )
