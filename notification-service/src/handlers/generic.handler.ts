import { publishNotification } from '../utils/notifications';
import { EventEnvelope, registerHandler } from '../libs/kafka-consumer';

export interface NotificationRequestedPayload {
    recipient_id: string;
    type: string;
    data: Record<string, any>;
}

export const setupGenericHandler = () => {
    registerHandler('courses.notification.requested.v1', async (envelope: EventEnvelope<NotificationRequestedPayload>) => {
        const { recipient_id, type, data } = envelope.payload;

        console.log(`[Kafka Handler] Processing notification request for user ${recipient_id}, type: ${type}`);

        await publishNotification(recipient_id, {
            type,
            ...data
        });
    });
};
