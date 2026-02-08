import { publishNotification } from '../utils/notifications';
import { EventEnvelope, registerHandler } from '../libs/kafka-consumer';

const COURSES_SERVICE_URL = process.env.COURSES_SERVICE_URL || 'http://localhost:8085';

interface LessonPayload {
    lesson_id: string;
    course_id: string;
    lesson_title?: string;
}

async function getCourseParticipants(courseId: string): Promise<string[]> {
    try {
        const response = await fetch(`${COURSES_SERVICE_URL}/api/v1/courses/${courseId}/enrollments`);
        if (!response.ok) return [];

        const body = await response.json();
        if (body.success && Array.isArray(body.data)) {
            return body.data.map((e: any) => e.student_id);
        }
        return [];
    } catch (error) {
        console.error(`[LessonHandler] Error fetching entries for course ${courseId}:`, error);
        return [];
    }
}

export const setupLessonHandlers = () => {
    // Lesson Started
    registerHandler('courses.lesson.started.v1', async (envelope: EventEnvelope<LessonPayload>) => {
        const { course_id, lesson_id } = envelope.payload;
        const studentIds = await getCourseParticipants(course_id);

        console.log(`[LessonHandler] Fanning out LESSON_STARTED to ${studentIds.length} students`);

        for (const studentId of studentIds) {
            await publishNotification(studentId, {
                type: 'LESSON_STARTED',
                lesson_id,
                course_id
            });
        }
    });

    // Lesson Canceled
    registerHandler('courses.lesson.canceled.v1', async (envelope: EventEnvelope<LessonPayload>) => {
        const { course_id, lesson_id } = envelope.payload;
        const studentIds = await getCourseParticipants(course_id);

        for (const studentId of studentIds) {
            await publishNotification(studentId, {
                type: 'LESSON_CANCELED',
                lesson_id,
                course_id
            });
        }
    });

    // Lesson Rescheduled
    registerHandler('courses.lesson.rescheduled.v1', async (envelope: EventEnvelope<any>) => {
        const { lesson_id } = envelope.payload;
        // We don't have course_id in payload, but aggregateID is lesson_id
        // For now, let's assume we need to fetch lesson details to get course_id
        // OR we could have added course_id to payload in courses-service
    });
};
