import redis from "../libs/redis";

// Redis cooldown constants for email verification
const EMAIL_VERIFICATION_COOLDOWN_SEC = parseInt(process.env.EMAIL_VERIFICATION_COOLDOWN_SEC || "900", 10); // 15 minutes
const EMAIL_VERIFICATION_LONG_COOLDOWN_SEC = parseInt(process.env.EMAIL_VERIFICATION_LONG_COOLDOWN_SEC || "3600", 10); // 60 minutes
const EMAIL_VERIFICATION_MAX_ATTEMPTS = parseInt(process.env.EMAIL_VERIFICATION_MAX_ATTEMPTS || "5", 10); // 5 attempts

/**
 * Redis key generators for email verification cooldowns and attempts
 */
function getEmailVerificationCooldownKey(email: string): string {
  return `email_verification_cooldown:${email}`;
}

function getEmailVerificationAttemptsKey(email: string): string {
  return `email_verification_attempts:${email}`;
}

/**
 * Check if email verification attempt is in cooldown
 * @param email - User email
 * @returns Promise<number> - Remaining cooldown seconds, 0 if no cooldown
 */
export async function checkEmailVerificationCooldown(email: string): Promise<number> {
  const key = getEmailVerificationCooldownKey(email);
  const remaining = await redis.ttl(key);
  return remaining > 0 ? remaining : 0;
}

/**
 * Get email verification attempt count
 * @param email - User email
 * @returns Promise<number> - Current attempt count
 */
async function getEmailVerificationAttempts(email: string): Promise<number> {
  const key = getEmailVerificationAttemptsKey(email);
  const attempts = await redis.get(key);
  return attempts ? parseInt(attempts, 10) : 0;
}

/**
 * Increment email verification attempt count
 * @param email - User email
 * @returns Promise<number> - New attempt count
 */
async function incrementEmailVerificationAttempts(email: string): Promise<number> {
  const key = getEmailVerificationAttemptsKey(email);
  const attempts = await redis.incr(key);
  
  // Set expiration on first attempt (24 hours)
  if (attempts === 1) {
    await redis.expire(key, 86400); // 24 hours
  }
  
  return attempts;
}

/**
 * Reset email verification attempt count
 * @param email - User email
 * @returns Promise<void>
 */
async function resetEmailVerificationAttempts(email: string): Promise<void> {
  const key = getEmailVerificationAttemptsKey(email);
  await redis.del(key);
}

/**
 * Set email verification cooldown with progressive cooldown based on attempts
 * @param email - User email
 * @returns Promise<number> - Cooldown duration in seconds that was set (0 if no cooldown)
 */
export async function setEmailVerificationCooldown(email: string): Promise<number> {
  const attempts = await incrementEmailVerificationAttempts(email);
  const cooldownKey = getEmailVerificationCooldownKey(email);
  
  let cooldownDuration: number;
  
  // If attempts exceed limit, apply progressive cooldown
  if (attempts >= EMAIL_VERIFICATION_MAX_ATTEMPTS) {
    // Check if user already hit cooldown before (for progressive increase)
    const existingCooldown = await redis.get(cooldownKey);
    if (existingCooldown) {
      // User hit cooldown before, apply longer cooldown
      cooldownDuration = EMAIL_VERIFICATION_LONG_COOLDOWN_SEC;
    } else {
      // First time exceeding limit, apply standard cooldown
      cooldownDuration = EMAIL_VERIFICATION_COOLDOWN_SEC;
    }
    
    await redis.set(cooldownKey, "1", "EX", cooldownDuration);
    return cooldownDuration;
  }
  
  // Within limit, no cooldown needed
  return 0;
}

/**
 * Check if email verification attempt should be allowed (checks both cooldown and attempts)
 * @param email - User email
 * @returns Promise<{ allowed: boolean; remainingCooldown: number; attempts: number }>
 */
export async function checkEmailVerificationAllowed(email: string): Promise<{
  allowed: boolean;
  remainingCooldown: number;
  attempts: number;
}> {
  const remainingCooldown = await checkEmailVerificationCooldown(email);
  const attempts = await getEmailVerificationAttempts(email);
  
  // If in cooldown, not allowed
  if (remainingCooldown > 0) {
    return { allowed: false, remainingCooldown, attempts };
  }
  
  // If attempts exceed limit and not in cooldown, need to set cooldown
  if (attempts >= EMAIL_VERIFICATION_MAX_ATTEMPTS) {
    return { allowed: false, remainingCooldown: 0, attempts };
  }
  
  return { allowed: true, remainingCooldown: 0, attempts };
}

/**
 * Clear email verification cooldown and reset attempts
 * @param email - User email
 * @returns Promise<void>
 */
export async function clearEmailVerificationCooldown(email: string): Promise<void> {
  const cooldownKey = getEmailVerificationCooldownKey(email);
  await redis.del(cooldownKey);
  await resetEmailVerificationAttempts(email);
}

