import kafka from './kafka';
import { EachMessageHandler } from 'kafkajs';

export const TOPICS = [
    'courses.lesson.started.v1',
    'courses.lesson.ended.v1',
    'courses.lesson.canceled.v1',
    'courses.lesson.rescheduled.v1',
    'courses.attendance.recorded.v1',
    'courses.attendance.finalized.v1',
    'courses.absence.requested.v1',
    'courses.absence.reviewed.v1',
    'courses.progress.updated.v1',
    'courses.notification.requested.v1',
    'courses.lesson.reminder.v1',
    'notifications.v1'
];

export interface EventEnvelope<T = any> {
    event_id: string;
    event_type: string;
    occurred_at: string;
    aggregate_id: string;
    actor_user_id: string;
    payload: T;
}

const handlers: Record<string, (envelope: EventEnvelope) => Promise<void>> = {};

export const registerHandler = (eventType: string, handler: (envelope: EventEnvelope) => Promise<void>) => {
    handlers[eventType] = handler;
};

const consumer = kafka.consumer({ groupId: process.env.KAFKA_GROUP_ID || 'notification-service' });

export const initConsumer = async () => {
    await consumer.connect();
    console.log('[Kafka] Registered handlers:', Object.keys(handlers));

    for (const topic of TOPICS) {
        await consumer.subscribe({ topic, fromBeginning: false });
    }

    await consumer.run({
        eachMessage: async ({ topic, partition, message }) => {
            if (!message.value) return;

            try {
                const rawData = JSON.parse(message.value.toString());
                
                // If the message doesn't have an event_type, we can't route it
                const eventType = rawData.event_type || rawData.eventType;
                if (!eventType) {
                    console.warn(`[Kafka] Received message without event_type on topic ${topic}`);
                    return;
                }

                // Construct a normalized envelope if it's missing fields
                const envelope: EventEnvelope = {
                    event_id: rawData.event_id || rawData.eventId || 'raw-' + Date.now(),
                    event_type: eventType,
                    occurred_at: rawData.occurred_at || rawData.occurredAt || new Date().toISOString(),
                    aggregate_id: rawData.aggregate_id || rawData.aggregateId || '',
                    actor_user_id: rawData.actor_user_id || rawData.actorUserId || '',
                    payload: rawData.payload || rawData // Use the whole object as payload if no 'payload' field
                };

                console.log(`[Kafka] Received event: ${envelope.event_type} on topic: ${topic}`);

                const handler = handlers[envelope.event_type];
                if (handler) {
                    await handler(envelope);
                } else {
                    console.warn(`[Kafka] No handler registered for event type: ${envelope.event_type}`);
                }
            } catch (error) {
                console.error(`[Kafka] Error processing message on topic ${topic}:`, error);
            }
        },
    });
};

export const stopConsumer = async () => {
    await consumer.disconnect();
};
