from fastapi import APIRouter, Depends, HTTPException, Response, BackgroundTasks
from app.api.dependencies import get_current_user
from app.schemas.chat import ResponseModel
from app.services.report_engine import report_engine
from app.models.database import get_db
from app.models.report import StudentReport
from sqlalchemy.orm import Session
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

@router.get("/parent/student/{student_id}/history")
async def get_report_history(
    student_id: str,
    db: Session = Depends(get_db),
    current_user: dict = Depends(get_current_user)
):
    """Returns a list of all past AI reports for a specific student."""
    if current_user["role"] != "PARENT":
        raise HTTPException(status_code=403, detail="Unauthorized")

    reports = db.query(StudentReport).filter(
        StudentReport.student_id == student_id,
        StudentReport.parent_id == current_user["user_id"]
    ).order_by(StudentReport.created_at.desc()).all()

    return {
        "success": True,
        "data": [
            {
                "id": r.id,
                "studentName": r.student_name,
                "period": r.period,
                "language": r.language,
                "createdAt": r.created_at,
                "summary": r.report_text[:100] + "..."
            } for r in reports
        ]
    }

@router.get("/history/{report_id}/download")
async def download_historical_report(
    report_id: str,
    db: Session = Depends(get_db),
    current_user: dict = Depends(get_current_user)
):
    """Generates a PDF for a specific past report from the database."""
    if current_user["role"] != "PARENT":
        raise HTTPException(status_code=403, detail="Unauthorized")

    report = db.query(StudentReport).filter(
        StudentReport.id == report_id,
        StudentReport.parent_id == current_user["user_id"]
    ).first()

    if not report:
        raise HTTPException(status_code=404, detail="Report not found")

    # Re-generate PDF from the saved text
    pdf_bytes = report_engine.generate_pdf(
        report.student_name, 
        report.report_text, 
        report.language, 
        report.period
    )

    return Response(
        content=pdf_bytes,
        media_type="application/pdf",
        headers={
            "Content-Disposition": f"attachment; filename=history_report_{report.student_id}_{report.period}.pdf"
        }
    )

@router.get("/parent/student/{student_id}/download")
async def download_parent_report(
    student_id: str,
    student_name: str = "Omar",
    language: str = "en",
    period: str = "weekly",
    current_user: dict = Depends(get_current_user)
):
    """Generates and returns a FRESH progress report as a downloadable PDF."""
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
