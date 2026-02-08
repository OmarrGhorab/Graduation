import { publishNotification } from '../utils/notifications';
import { EventEnvelope, registerHandler } from '../libs/kafka-consumer';

export const setupAttendanceHandlers = () => {
    // Attendance Recorded (Optional: notify student or parent)
    registerHandler('courses.attendance.recorded.v1', async (envelope: EventEnvelope<any>) => {
        const { student_id, status, lesson_id } = envelope.payload;

        console.log(`[AttendanceHandler] Attendance recorded for ${student_id}: ${status}`);

        // Example: Notify student their attendance was marked
        await publishNotification(student_id, {
            type: 'ATTENDANCE_RECORDED',
            status,
            lesson_id
        });
    });

    // Progress Updated
    registerHandler('courses.progress.updated.v1', async (envelope: EventEnvelope<any>) => {
        const { student_id, overall_progress, course_id } = envelope.payload;

        await publishNotification(student_id, {
            type: 'PROGRESS_UPDATED',
            overall_progress,
            course_id
        });
    });

    // Absence Requested
    registerHandler('courses.absence.requested.v1', async (envelope: EventEnvelope<any>) => {
        // Notify teacher/admin?
        // actor_user_id is the student
        // aggregateID is the requestID
    });

    // Absence Reviewed
    registerHandler('courses.absence.reviewed.v1', async (envelope: EventEnvelope<any>) => {
        // Notify student of result
        const { student_id } = envelope.payload; // Need to ensure student_id is in payload
        // If not, we might need actor_user_id logic
    });
};
