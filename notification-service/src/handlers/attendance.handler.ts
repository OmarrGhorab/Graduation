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
        const { student_id } = envelope.payload; 
    });

    // Attendance Recorded (Offline Scan or Mark Absent)
    registerHandler('courses.attendance.recorded.v1', async (envelope: EventEnvelope<any>) => {
        const { 
            lesson_id, lesson_title, course_id, course_title, 
            student_id, status 
        } = envelope.payload;

        // Notify parents of the student's status
        const parents = await getParentsOfStudent(student_id);
        const statusText = status.toLowerCase(); // present, late, absent
        
        let body = `Notification: Your child was marked ${statusText} for the lesson "${lesson_title}" in ${course_title}.`;
        if (status === 'ABSENT' || status === 'LATE') {
            body += ` Please check the app if you wish to submit an excuse or appeal.`;
        }

        for (const parent of parents) {
            await publishNotification(parent.id, {
                type: 'ATTENDANCE_STATUS_UPDATE',
                title: `Attendance: ${status}`,
                body: body,
                data: { lesson_id, course_id, student_id, status }
            });
        }
    });

    // Absence Requested (Appeal/Excuse)
    registerHandler('courses.absence.requested.v1', async (envelope: EventEnvelope<any>) => {
        const { 
            request_id, lesson_id, lesson_title, course_id, course_title, 
            student_id, teacher_id, reason 
        } = envelope.payload;

        // 1. Notify the Teacher about the new request
        await publishNotification(teacher_id, {
            type: 'ABSENCE_REQUEST_TEACHER',
            title: 'New Absence Excuse Submitted',
            body: `A student has submitted an excuse for "${lesson_title}". Reason: ${reason}`,
            data: { request_id, lesson_id, course_id, student_id }
        });

        // 2. Notify the Parents that the excuse was submitted
        const parents = await getParentsOfStudent(student_id);
        for (const parent of parents) {
            await publishNotification(parent.id, {
                type: 'ABSENCE_REQUEST_PARENT',
                title: 'Absence Excuse Sent',
                body: `An excuse for the lesson "${lesson_title}" has been submitted for review.`,
                data: { request_id, lesson_id, course_id }
            });
        }
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
            body: `Alert: A potential cheating attempt was detected in "${course_title}". A student tried to scan for attendance using a device that was already used by another student in this lesson.`,
            data: { lesson_id, course_id, student_id, existing_student_id, device_id }
        });

        // 2. Notify parents of the student who tried to scan
        const parentsA = await getParentsOfStudent(student_id);
        for (const parent of parentsA) {
            await publishNotification(parent.id, {
                type: 'ATTENDANCE_FRAUD_PARENT',
                title: 'Security Alert: Attendance Issue',
                body: `We detected an attendance scanning issue for your child in "${course_title}". Multiple accounts were used on the same device. Please ensure your child uses their own device.`,
                data: { lesson_id, course_id, student_id }
            });
        }

        // 3. Notify parents of the student who was already scanned (optional, but good for context)
        if (student_id !== existing_student_id) {
            const parentsB = await getParentsOfStudent(existing_student_id);
            for (const parent of parentsB) {
                await publishNotification(parent.id, {
                    type: 'ATTENDANCE_FRAUD_PARENT',
                    title: 'Security Alert: Attendance Issue',
                    body: `Your child's device was used by another student to scan for attendance in "${course_title}". This is not allowed for security reasons.`,
                    data: { lesson_id, course_id, student_id: existing_student_id }
                });
            }
        }
    });

    // Lesson Video Ready (Background Processing Finished)
    registerHandler('courses.lesson.video_ready.v1', async (envelope: EventEnvelope<any>) => {
        const { lesson_id, lesson_title, teacher_id } = envelope.payload;

        await publishNotification(teacher_id, {
            type: 'VIDEO_READY',
            title: 'Video Upload Complete',
            body: `The video for your lesson "${lesson_title}" has been processed and is now available for students.`,
            data: { lesson_id }
        });
    });

    // Lesson Video Failed
    registerHandler('courses.lesson.video_failed.v1', async (envelope: EventEnvelope<any>) => {
        const { lesson_id, lesson_title, teacher_id, error } = envelope.payload;

        await publishNotification(teacher_id, {
            type: 'VIDEO_FAILED',
            title: 'Video Processing Failed',
            body: `Unfortunately, the video for "${lesson_title}" could not be processed. Error: ${error}`,
            data: { lesson_id }
        });
    });
};

/**
 * Internal helper to fetch parents from auth-service
 */
async function getParentsOfStudent(userId: string): Promise<any[]> {
    const AUTH_SERVICE_URL = process.env.AUTH_SERVICE_URL || "http://localhost:6001";
    const INTERNAL_SERVICE_SECRET = process.env.INTERNAL_SERVICE_SECRET || "";
    
    try {
        const response = await fetch(`${AUTH_SERVICE_URL}/api/v1/internal/users/${userId}/parents`, {
            headers: {
                "x-internal-service-secret": INTERNAL_SERVICE_SECRET,
            },
        });

        if (!response.ok) return [];
        const result = await response.json();
        return result.data || [];
    } catch (error) {
        console.error(`[AttendanceHandler] Error fetching parents for ${userId}:`, error);
        return [];
    }
}
