import Redis from "ioredis"
import dotenv from "dotenv";

dotenv.config();

const REDIS_URL = process.env.REDIS_URL || "";

const client = new Redis(REDIS_URL);

// Handle Redis connection events
client.on("connect", () => {
  console.log("[Redis] Connected successfully");
});

client.on("error", (error) => {
  console.error("[Redis] Connection error:", error);
});

client.on("ready", () => {
  console.log("[Redis] Ready to accept commands");
});

export default client;