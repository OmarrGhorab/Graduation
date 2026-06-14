import { publishNotification } from '../utils/notifications';
import { EventEnvelope, registerHandler } from '../libs/kafka-consumer';

export const setupAttendanceHandlers = () => {

    // Attendance Recorded — single consolidated handler: notifies student AND parents
    registerHandler('courses.attendance.recorded.v1', async (envelope: EventEnvelope<any>) => {
        const {
            student_id, lesson_id, lesson_title, course_id, course_title, status
        } = envelope.payload;

        console.log(`[AttendanceHandler] Attendance recorded for ${student_id}: ${status}`);

        // 1. Notify the student their attendance was marked
        await publishNotification(student_id, {
            type: 'ATTENDANCE_RECORDED',
            status,
            lesson_id,
            course_id,
            lesson_title,
            course_title,
        });

        // 2. Notify parents
        const parents = await getParentsOfStudent(student_id);
        const statusText = (status as string).toLowerCase();
        let parentBody = `Your child was marked ${statusText} for "${lesson_title || 'a lesson'}" in ${course_title || 'a course'}.`;
        if (status === 'ABSENT' || status === 'LATE') {
            parentBody += ' You can submit an excuse or appeal from the app.';
        }

        for (const parent of parents) {
            await publishNotification(parent.id, {
                type: 'CHILD_ATTENDANCE_RECORDED',
                lesson_id,
                course_id,
                child_id: student_id,
                lesson_title,
                course_title,
                status,
                title: `Attendance: ${status}`,
                body: parentBody,
            });
        }
    });

    // Attendance Finalized — lesson ended, all absent students already notified via
    // individual courses.attendance.recorded.v1 events from markAbsentStudents().
    registerHandler('courses.attendance.finalized.v1', async (envelope: EventEnvelope<any>) => {
        const { lesson_id } = envelope.payload;
        console.log(`[AttendanceHandler] Attendance finalized for lesson ${lesson_id}`);
    });

    // Progress Updated
    registerHandler('courses.progress.updated.v1', async (envelope: EventEnvelope<any>) => {
        const { student_id, overall_progress, course_id } = envelope.payload;

        await publishNotification(student_id, {
            type: 'PROGRESS_UPDATED',
            overall_progress,
            course_id,
        });
    });

    // Absence Requested (Appeal/Excuse) — single registration
    registerHandler('courses.absence.requested.v1', async (envelope: EventEnvelope<any>) => {
        const {
            request_id, lesson_id, lesson_title, course_id, course_title,
            student_id, teacher_id, reason
        } = envelope.payload;

        // 1. Notify teacher
        await publishNotification(teacher_id, {
            type: 'ABSENCE_REQUEST_TEACHER',
            title: 'New Absence Excuse Submitted',
            body: `A student submitted an excuse for "${lesson_title}". Reason: ${reason}`,
            data: { request_id, lesson_id, course_id, student_id },
        });

        // 2. Notify parents
        const parents = await getParentsOfStudent(student_id);
        for (const parent of parents) {
            await publishNotification(parent.id, {
                type: 'ABSENCE_REQUEST_PARENT',
                title: 'Absence Excuse Sent',
                body: `An excuse for "${lesson_title}" has been submitted for review.`,
                data: { request_id, lesson_id, course_id },
            });
        }
    });

    // Absence Reviewed — notify student of the outcome, and CC parents
    registerHandler('courses.absence.reviewed.v1', async (envelope: EventEnvelope<any>) => {
        const {
            request_id, student_id, lesson_id, course_id,
            lesson_title, status, response_note
        } = envelope.payload;

        if (!student_id) {
            console.warn('[AttendanceHandler] courses.absence.reviewed.v1 missing student_id — skipping');
            return;
        }

        const isApproved = (status as string).toUpperCase() === 'APPROVED';
        const statusLabel = isApproved ? 'approved' : 'rejected';
        const lessonLabel = lesson_title || 'a lesson';

        const studentTitle = isApproved
            ? 'Absence Excuse Approved ✅'
            : 'Absence Excuse Rejected';
        const studentBody = isApproved
            ? `Your absence excuse for "${lessonLabel}" has been approved. Your attendance has been updated to Excused.`
            : `Your absence excuse for "${lessonLabel}" was not approved.${response_note ? ` Note: ${response_note}` : ''}`;

        // 1. Notify the student
        await publishNotification(student_id, {
            type: 'ATTENDANCE_STATUS_UPDATE',
            title: studentTitle,
            body: studentBody,
            data: { request_id, lesson_id, course_id, status },
        });

        // 2. Notify parents
        const parents = await getParentsOfStudent(student_id);
        const parentBody = isApproved
            ? `Your child's absence excuse for "${lessonLabel}" has been approved.`
            : `Your child's absence excuse for "${lessonLabel}" was not approved.${response_note ? ` Note: ${response_note}` : ''}`;

        for (const parent of parents) {
            await publishNotification(parent.id, {
                type: 'ABSENCE_REQUEST_PARENT',
                title: `Child Absence Excuse ${isApproved ? 'Approved' : 'Rejected'}`,
                body: parentBody,
                data: { request_id, lesson_id, course_id, status, child_id: student_id },
            });
        }

        console.log(`[AttendanceHandler] Absence ${statusLabel} notification sent to student ${student_id} and ${parents.length} parent(s)`);
    });

    // Attendance Fraud Detected
    registerHandler('courses.attendance.fraud_detected.v1', async (envelope: EventEnvelope<any>) => {
        const {
            lesson_id, lesson_title, course_id, course_title,
            student_id, existing_student_id, device_id, teacher_id
        } = envelope.payload;

        console.log(`[AttendanceHandler] FRAUD DETECTED in lesson ${lesson_id}. Device ${device_id} shared by ${student_id} and ${existing_student_id}`);

        // 1. Notify the teacher
        await publishNotification(teacher_id, {
            type: 'ATTENDANCE_FRAUD_TEACHER',
            title: 'Attendance Fraud Warning',
            body: `A potential cheating attempt was detected in "${course_title}". A student tried to scan using a device already used by another student.`,
            data: { lesson_id, course_id, student_id, existing_student_id, device_id },
        });

        // 2. Notify parents of the student who tried to scan
        const parentsA = await getParentsOfStudent(student_id);
        for (const parent of parentsA) {
            await publishNotification(parent.id, {
                type: 'ATTENDANCE_FRAUD_PARENT',
                title: 'Security Alert: Attendance Issue',
                body: `We detected an attendance scanning issue for your child in "${course_title}". Multiple accounts were used on the same device.`,
                data: { lesson_id, course_id, student_id },
            });
        }

        // 3. Notify parents of the student whose device was used
        if (student_id !== existing_student_id) {
            const parentsB = await getParentsOfStudent(existing_student_id);
            for (const parent of parentsB) {
                await publishNotification(parent.id, {
                    type: 'ATTENDANCE_FRAUD_PARENT',
                    title: 'Security Alert: Attendance Issue',
                    body: `Your child's device was used by another student to scan for attendance in "${course_title}".`,
                    data: { lesson_id, course_id, student_id: existing_student_id },
                });
            }
        }
    });

    // Lesson Video Ready
    registerHandler('courses.lesson.video_ready.v1', async (envelope: EventEnvelope<any>) => {
        const { lesson_id, lesson_title, teacher_id } = envelope.payload;

        await publishNotification(teacher_id, {
            type: 'VIDEO_READY',
            title: 'Video Upload Complete',
            body: `The video for your lesson "${lesson_title}" has been processed and is now available for students.`,
            data: { lesson_id },
        });
    });

    // Lesson Video Failed
    registerHandler('courses.lesson.video_failed.v1', async (envelope: EventEnvelope<any>) => {
        const { lesson_id, lesson_title, teacher_id, error } = envelope.payload;

        await publishNotification(teacher_id, {
            type: 'VIDEO_FAILED',
            title: 'Video Processing Failed',
            body: `The video for "${lesson_title}" could not be processed. Error: ${error}`,
            data: { lesson_id },
        });
    });
};

async function getParentsOfStudent(userId: string): Promise<any[]> {
    const AUTH_SERVICE_URL = process.env.AUTH_SERVICE_URL || 'http://localhost:6001';
    const INTERNAL_SERVICE_SECRET = process.env.INTERNAL_SERVICE_SECRET || '';

    try {
        const response = await fetch(`${AUTH_SERVICE_URL}/api/v1/internal/users/${userId}/parents`, {
            headers: { 'x-internal-service-secret': INTERNAL_SERVICE_SECRET },
        });
        if (!response.ok) return [];
        const result = await response.json();
        return result.data || [];
    } catch (error) {
        console.error(`[AttendanceHandler] Error fetching parents for ${userId}:`, error);
        return [];
    }
}
