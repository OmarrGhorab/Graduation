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

export async function createAndStoreOtp(target: string, otp?: string): Promise<string> {
  const k = keys(target);
  const code = otp || generateNumericOtp();
  await redis.set(k.value, code, "EX", OTP_TTL_SEC);
  // reset attempts on new OTP
  await redis.del(k.attempts);
  return code;
}

export async function verifyOtp(target: string, code: string): Promise<boolean> {
  const k = keys(target);

  // cooldown check
  const isCooling = await redis.get(k.cooldown);
  if (isCooling) return false;

  const stored = await redis.get(k.value);
  if (!stored) return false;

  const match = stored === code;
  if (!match) {
    const attempts = await redis.incr(k.attempts);
    if (attempts === 1) {
      await redis.expire(k.attempts, OTP_TTL_SEC);
    }
    if (attempts >= OTP_ATTEMPT_LIMIT) {
      await redis.set(k.cooldown, "1", "EX", OTP_COOLDOWN_SEC);
      await redis.del(k.attempts);
    }
    return false;
  }

  // success: cleanup
  await redis.del(k.value);
  await redis.del(k.attempts);
  return true;
}

export async function verifyOtpWithoutConsuming(target: string, code: string): Promise<boolean> {
  const k = keys(target);

  // cooldown check
  const isCooling = await redis.get(k.cooldown);
  if (isCooling) return false;

  const stored = await redis.get(k.value);
  if (!stored) return false;

  const match = stored === code;
  if (!match) {
    const attempts = await redis.incr(k.attempts);
    if (attempts === 1) {
      await redis.expire(k.attempts, OTP_TTL_SEC);
    }
    if (attempts >= OTP_ATTEMPT_LIMIT) {
      await redis.set(k.cooldown, "1", "EX", OTP_COOLDOWN_SEC);
      await redis.del(k.attempts);
    }
    return false;
  }

  // success: do NOT delete OTP, just return true
  return true;
}


