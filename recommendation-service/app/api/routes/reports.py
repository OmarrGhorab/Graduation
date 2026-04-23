from fastapi import APIRouter, Depends, HTTPException, Response, BackgroundTasks
from app.api.dependencies import get_current_user
from app.schemas.chat import ResponseModel
from app.services.report_engine import report_engine
import logging

logger = logging.getLogger(__name__)

router = APIRouter()

@router.post("/parent/student/{student_id}/trigger", response_model=ResponseModel)
async def trigger_parent_report(
    student_id: str,
    background_tasks: BackgroundTasks,
    student_name: str = "Omar", 
    language: str = "en",
    period: str = "weekly",
    current_user: dict = Depends(get_current_user)
):
    """
    Manually triggers the generation of an AI progress report for a student.
    Runs in the background and notifies the parent via FCM when finished.
    """
    if current_user["role"] != "PARENT":
        raise HTTPException(status_code=403, detail="Only parents can trigger progress reports.")

    # Start generation in background
    background_tasks.add_task(
        report_engine.run_report_background,
        current_user["user_id"],
        student_id,
        student_name,
        language,
        period
    )

    return {
        "success": True,
        "message": "Report generation started in the background. You will be notified when it's ready.",
        "data": {
            "studentId": student_id,
            "period": period,
            "status": "processing"
        }
    }

@router.get("/parent/student/{student_id}/download")
async def download_parent_report(
    student_id: str,
    student_name: str = "Omar",
    language: str = "en",
    period: str = "weekly",
    current_user: dict = Depends(get_current_user)
):
    """Generates and returns the progress report as a downloadable PDF."""
    if current_user["role"] != "PARENT":
         raise HTTPException(status_code=403, detail="Unauthorized")

    # 1. Generate the AI text first
    report_text = await report_engine.generate_parent_report(student_id, student_name, language, period)
    if not report_text:
        raise HTTPException(status_code=500, detail="Failed to generate report text.")

    # 2. Convert to PDF
    pdf_bytes = report_engine.generate_pdf(student_name, report_text, language, period)

    # 3. Return as a file response
    return Response(
        content=pdf_bytes,
        media_type="application/pdf",
        headers={
            "Content-Disposition": f"attachment; filename=progress_report_{student_id}_{period}.pdf"
        }
    )
