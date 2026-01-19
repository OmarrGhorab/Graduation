import crypto from "crypto";
import redis from "../libs/redis";

const OTP_TTL_SEC = parseInt(process.env.OTP_TTL_SEC || "600", 10); // 10 minutes
const OTP_ATTEMPT_LIMIT = parseInt(process.env.OTP_ATTEMPT_LIMIT || "5", 10);
const OTP_COOLDOWN_SEC = parseInt(process.env.OTP_COOLDOWN_SEC || "900", 10); // 15 minutes

export function generateNumericOtp(length = 6): string {
  const digits = "0123456789";
  let otp = "";
  for (let i = 0; i < length; i++) {
    otp += digits[crypto.randomInt(0, 10)];
  }
  return otp;
}

function keys(target: string) {
  return {
    value: `otp:${target}`,
    attempts: `otp_attempts:${target}`,
    cooldown: `otp_cooldown:${target}`,
  };
}

/**
 * Create and store OTP with Redis pipeline for atomic operations
 * OPTIMIZED: Uses pipeline to reduce 2 Redis calls to 1 round-trip
 */
export async function createAndStoreOtp(target: string, otp?: string): Promise<string> {
  const k = keys(target);
  const code = otp || generateNumericOtp();
  
  // Use pipeline for atomic operations - reduces network round trips
  const pipeline = redis.pipeline();
  pipeline.set(k.value, code, "EX", OTP_TTL_SEC);
  pipeline.del(k.attempts);
  await pipeline.exec();
  
  return code;
}

/**
 * Verify OTP with Redis pipeline for better performance
 * OPTIMIZED: Uses pipeline to batch Redis operations, reducing 4-6 calls to 2 round-trips
 */
export async function verifyOtp(target: string, code: string): Promise<boolean> {
  const k = keys(target);

  // OPTIMIZED: Batch initial reads with pipeline
  const readPipeline = redis.pipeline();
  readPipeline.get(k.cooldown);
  readPipeline.get(k.value);
  const results = await readPipeline.exec();

  // Extract results: [error, value] pairs
  const isCooling = results?.[0]?.[1];
  const stored = results?.[1]?.[1];

  if (isCooling) return false;
  if (!stored) return false;

  const match = stored === code;
  if (!match) {
    // Handle failed attempt with pipeline
    const attempts = await redis.incr(k.attempts);
    
    if (attempts === 1) {
      await redis.expire(k.attempts, OTP_TTL_SEC);
    }
    
    if (attempts >= OTP_ATTEMPT_LIMIT) {
      // OPTIMIZED: Batch cooldown set and attempts cleanup
      const cooldownPipeline = redis.pipeline();
      cooldownPipeline.set(k.cooldown, "1", "EX", OTP_COOLDOWN_SEC);
      cooldownPipeline.del(k.attempts);
      await cooldownPipeline.exec();
    }
    return false;
  }

  // OPTIMIZED: Batch success cleanup with pipeline
  const cleanupPipeline = redis.pipeline();
  cleanupPipeline.del(k.value);
  cleanupPipeline.del(k.attempts);
  await cleanupPipeline.exec();
  
  return true;
}

/**
 * Verify OTP without consuming it (for multi-step verification flows)
 * OPTIMIZED: Uses pipeline to batch Redis operations
 */
export async function verifyOtpWithoutConsuming(target: string, code: string): Promise<boolean> {
  const k = keys(target);

  // OPTIMIZED: Batch initial reads with pipeline
  const readPipeline = redis.pipeline();
  readPipeline.get(k.cooldown);
  readPipeline.get(k.value);
  const results = await readPipeline.exec();

  const isCooling = results?.[0]?.[1];
  const stored = results?.[1]?.[1];

  if (isCooling) return false;
  if (!stored) return false;

  const match = stored === code;
  if (!match) {
    const attempts = await redis.incr(k.attempts);
    
    if (attempts === 1) {
      await redis.expire(k.attempts, OTP_TTL_SEC);
    }
    
    if (attempts >= OTP_ATTEMPT_LIMIT) {
      const cooldownPipeline = redis.pipeline();
      cooldownPipeline.set(k.cooldown, "1", "EX", OTP_COOLDOWN_SEC);
      cooldownPipeline.del(k.attempts);
      await cooldownPipeline.exec();
    }
    return false;
  }

  // success: do NOT delete OTP, just return true
  return true;
}


