import Redis from "ioredis"
import dotenv from "dotenv";

dotenv.config();

const REDIS_URL = process.env.REDIS_URL || "";
const client = new Redis(REDIS_URL);

export default client;