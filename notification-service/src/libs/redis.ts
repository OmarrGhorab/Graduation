import Redis from "ioredis"
import dotenv from "dotenv";

dotenv.config();

const REDIS_URL = process.env.REDIS_URL || "redis://localhost:6379";
const client = new Redis(REDIS_URL);

// Handle Redis connection errors
client.on('error', (err) => {
  console.error('Redis connection error:', err);
});

client.on('connect', () => {
  console.log('Connected to Redis');
});

export default client;
