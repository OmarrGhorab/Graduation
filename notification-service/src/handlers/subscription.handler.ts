import { publishNotification } from '../utils/notifications';
import { EventEnvelope, registerHandler } from '../libs/kafka-consumer';

export const setupSubscriptionHandlers = () => {
    // Subscription Renewal Soon
    registerHandler('subscription.renewal_soon', async (envelope: EventEnvelope) => {
        const payload = envelope.payload;
        const userId = payload.user_id;

        console.log(`[SubscriptionHandler] Processing renewal reminder for user ${userId}`);

        await publishNotification(userId, {
            type: 'SUBSCRIPTION_RENEWAL_SOON',
            subscription_id: payload.subscription_id,
            course_id: payload.course_id,
            amount: payload.amount,
            currency: payload.currency,
            next_billing: payload.next_billing,
            days_left: payload.days_left
        });
    });

    // Subscription Payment Failed
    registerHandler('subscription.payment_failed', async (envelope: EventEnvelope) => {
        const payload = envelope.payload;
        const userId = payload.user_id;

        await publishNotification(userId, {
            type: 'SUBSCRIPTION_PAYMENT_FAILED',
            subscription_id: payload.subscription_id,
            reason: payload.reason
        });
    });
};
