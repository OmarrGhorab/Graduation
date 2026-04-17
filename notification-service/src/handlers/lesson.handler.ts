import { publishNotification, getChildParents } from '../utils/notifications';
import { EventEnvelope, registerHandler } from '../libs/kafka-consumer';

const COURSES_SERVICE_URL = process.env.COURSES_SERVICE_URL || 'http://localhost:8085';
const AUTH_SERVICE_URL = process.env.AUTH_SERVICE_URL || 'http://localhost:6001';
const INTERNAL_SERVICE_SECRET = process.env.INTERNAL_SERVICE_SECRET || '';

interface LessonPayload {
    lesson_id: string;
    course_id: string;
    lesson_title?: string;
}

async function getCourseParticipants(courseId: string): Promise<string[]> {
    try {
        const url = `${COURSES_SERVICE_URL}/api/v1/courses/${courseId}/enrollments`;
        const response = await fetch(url, {
            headers: {
                'x-internal-service-secret': process.env.INTERNAL_SERVICE_SECRET || ''
            }
        });
        
        if (!response.ok) {
            console.error(`[LessonHandler] Failed to fetch enrollments for course ${courseId}: ${response.status}`);
            return [];
        }

        const body = await response.json();
        if (body.success && Array.isArray(body.data)) {
            return body.data.map((e: any) => e.userId || e.student_id || e.studentId);
        }
        return [];
    } catch (error) {
        console.error(`[LessonHandler] Error fetching entries for course ${courseId}:`, error);
        return [];
    }
}

async function getChildInfo(childId: string): Promise<{ name: string } | null> {
    try {
        const response = await fetch(`${AUTH_SERVICE_URL}/api/v1/internal/users/${childId}`, {
            headers: { 'x-internal-service-secret': INTERNAL_SERVICE_SECRET }
        });
        const body = await response.json();
        return body.success ? { name: body.data.name } : null;
    } catch (error) {
        console.error(`[LessonHandler] Error fetching child info for ${childId}:`, error);
        return null;
    }
}

export const setupLessonHandlers = () => {
    // Lesson Started
    registerHandler('courses.lesson.started.v1', async (envelope: EventEnvelope<LessonPayload>) => {
        const { course_id, lesson_id, lesson_title } = envelope.payload;
        const studentIds = await getCourseParticipants(course_id);

        console.log(`[LessonHandler] Fanning out LESSON_STARTED to ${studentIds.length} students`);

        for (const studentId of studentIds) {
            // 1. Notify student
            await publishNotification(studentId, {
                type: 'LESSON_STARTED',
                lesson_id,
                course_id,
                lesson_title
            });

            // 2. Notify parents
            const childInfo = await getChildInfo(studentId);
            const parents = await getChildParents(studentId);
            for (const parent of parents) {
                await publishNotification(parent.id, {
                    type: 'CHILD_LESSON_STARTED',
                    lesson_id,
                    course_id,
                    child_id: studentId,
                    child_name: childInfo?.name || "Your child",
                    lesson_title
                });
            }
        }
    });

    // Lesson Canceled
    registerHandler('courses.lesson.canceled.v1', async (envelope: EventEnvelope<LessonPayload>) => {
        const { course_id, lesson_id, lesson_title } = envelope.payload;
        const studentIds = await getCourseParticipants(course_id);

        for (const studentId of studentIds) {
            // 1. Notify student
            await publishNotification(studentId, {
                type: 'LESSON_CANCELED',
                lesson_id,
                course_id,
                lesson_title
            });

            // 2. Notify parents
            const childInfo = await getChildInfo(studentId);
            const parents = await getChildParents(studentId);
            for (const parent of parents) {
                await publishNotification(parent.id, {
                    type: 'LESSON_CANCELED', // Re-using LESSON_CANCELED title/body but directed to parent
                    lesson_id,
                    course_id,
                    child_name: childInfo?.name || "Your child",
                    lesson_title
                });
            }
        }
    });

    // Lesson Ended
    registerHandler('courses.lesson.ended.v1', async (envelope: EventEnvelope<LessonPayload>) => {
        const { course_id, lesson_id, lesson_title } = envelope.payload;
        const studentIds = await getCourseParticipants(course_id);

        for (const studentId of studentIds) {
            // 1. Notify student
            await publishNotification(studentId, {
                type: 'LESSON_ENDED',
                lesson_id,
                course_id,
                lesson_title
            });

            // 2. Notify parents
            const childInfo = await getChildInfo(studentId);
            const parents = await getChildParents(studentId);
            for (const parent of parents) {
                await publishNotification(parent.id, {
                    type: 'CHILD_LESSON_ENDED',
                    lesson_id,
                    course_id,
                    child_id: studentId,
                    child_name: childInfo?.name || "Your child",
                    lesson_title
                });
            }
        }
    });

    // Lesson Rescheduled
    registerHandler('courses.lesson.rescheduled.v1', async (envelope: EventEnvelope<any>) => {
        const { course_id, lesson_id, lesson_title, new_scheduled_at } = envelope.payload;
        const studentIds = await getCourseParticipants(course_id);

        for (const studentId of studentIds) {
            // 1. Notify student
            await publishNotification(studentId, {
                type: 'LESSON_RESCHEDULED',
                lesson_id,
                course_id,
                lesson_title,
                new_scheduled_at
            });

            // 2. Notify parents
            const childInfo = await getChildInfo(studentId);
            const parents = await getChildParents(studentId);
            for (const parent of parents) {
                await publishNotification(parent.id, {
                    type: 'LESSON_RESCHEDULED',
                    lesson_id,
                    course_id,
                    child_name: childInfo?.name || "Your child",
                    lesson_title,
                    new_scheduled_at
                });
            }
        }
    });

    // Lesson Reminder
    registerHandler('courses.lesson.reminder.v1', async (envelope: EventEnvelope<any>) => {
        const { course_id, lesson_id, minutes_before, lesson_title } = envelope.payload;
        const studentIds = await getCourseParticipants(course_id);

        console.log(`[LessonHandler] Fanning out LESSON_REMINDER (${minutes_before}m) for ${lesson_id} to ${studentIds.length} students`);

        for (const studentId of studentIds) {
            await publishNotification(studentId, {
                type: 'LESSON_REMINDER',
                lesson_id,
                course_id,
                lesson_title,
                minutes_before
            });
        }
    });

    // Attendance Recorded (Notifying Parents)
    registerHandler('courses.attendance.recorded.v1', async (envelope: EventEnvelope<any>) => {
        const { student_id, lesson_id, lesson_title, course_id, status } = envelope.payload;
        
        // Notify parents
        const childInfo = await getChildInfo(student_id);
        const parents = await getChildParents(student_id);
        
        console.log(`[LessonHandler] Notifying ${parents.length} parents about child attendance for student ${student_id}`);
        
        for (const parent of parents) {
            await publishNotification(parent.id, {
                type: 'CHILD_ATTENDANCE_RECORDED',
                lesson_id,
                course_id,
                child_id: student_id,
                child_name: childInfo?.name || "Your child",
                lesson_title,
                status
            });
        }
    });
};
