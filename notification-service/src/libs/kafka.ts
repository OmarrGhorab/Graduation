import { Kafka } from 'kafkajs';

const kafka = new Kafka({
    clientId: 'notification-service',
    brokers: (process.env.KAFKA_BROKERS || 'localhost:9092').split(','),
    retry: {
        initialRetryTime: 100,
        retries: 8
    }
});

export default kafka;
