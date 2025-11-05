import redis from "../libs/redis";

// Redis cooldown constants for password reset
const FORGOT_PASSWORD_COOLDOWN_SEC = parseInt(process.env.FORGOT_PASSWORD_COOLDOWN_SEC || "300", 10); // 5 minutes
const FORGOT_PASSWORD_LONG_COOLDOWN_SEC = parseInt(process.env.FORGOT_PASSWORD_LONG_COOLDOWN_SEC || "1800", 10); // 30 minutes
const RESET_PASSWORD_COOLDOWN_SEC = parseInt(process.env.RESET_PASSWORD_COOLDOWN_SEC || "60", 10); // 1 minute
const RESET_PASSWORD_LONG_COOLDOWN_SEC = parseInt(process.env.RESET_PASSWORD_LONG_COOLDOWN_SEC || "1800", 10); // 30 minutes

// Attempt limits
const FORGOT_PASSWORD_MAX_ATTEMPTS = parseInt(process.env.FORGOT_PASSWORD_MAX_ATTEMPTS || "3", 10); // 3 attempts
const RESET_PASSWORD_MAX_ATTEMPTS = parseInt(process.env.RESET_PASSWORD_MAX_ATTEMPTS || "3", 10); // 3 attempts

/**
 * Redis key generators for password reset cooldowns and attempts
 */
function getForgotPasswordCooldownKey(email: string): string {
  return `forgot_password_cooldown:${email}`;
}

function getForgotPasswordAttemptsKey(email: string): string {
  return `forgot_password_attempts:${email}`;
}

function getResetPasswordCooldownKey(email: string): string {
  return `reset_password_cooldown:${email}`;
}

function getResetPasswordAttemptsKey(email: string): string {
  return `reset_password_attempts:${email}`;
}

/**
 * Check if forgot password request is in cooldown
 * @param email - User email
 * @returns Promise<number> - Remaining cooldown seconds, 0 if no cooldown
 */
export async function checkForgotPasswordCooldown(email: string): Promise<number> {
  const key = getForgotPasswordCooldownKey(email);
  const remaining = await redis.ttl(key);
  return remaining > 0 ? remaining : 0;
}

/**
 * Get forgot password attempt count
 * @param email - User email
 * @returns Promise<number> - Current attempt count
 */
async function getForgotPasswordAttempts(email: string): Promise<number> {
  const key = getForgotPasswordAttemptsKey(email);
  const attempts = await redis.get(key);
  return attempts ? parseInt(attempts, 10) : 0;
}

/**
 * Increment forgot password attempt count
 * @param email - User email
 * @returns Promise<number> - New attempt count
 */
async function incrementForgotPasswordAttempts(email: string): Promise<number> {
  const key = getForgotPasswordAttemptsKey(email);
  const attempts = await redis.incr(key);
  
  // Set expiration on first attempt (24 hours)
  if (attempts === 1) {
    await redis.expire(key, 86400); // 24 hours
  }
  
  return attempts;
}

/**
 * Reset forgot password attempt count
 * @param email - User email
 * @returns Promise<void>
 */
async function resetForgotPasswordAttempts(email: string): Promise<void> {
  const key = getForgotPasswordAttemptsKey(email);
  await redis.del(key);
}

/**
 * Set forgot password cooldown with progressive cooldown based on attempts
 * @param email - User email
 * @returns Promise<number> - Cooldown duration in seconds that was set
 */
export async function setForgotPasswordCooldown(email: string): Promise<number> {
  const attempts = await incrementForgotPasswordAttempts(email);
  const cooldownKey = getForgotPasswordCooldownKey(email);
  
  let cooldownDuration: number;
  
  // If attempts exceed limit, apply progressive cooldown
  if (attempts >= FORGOT_PASSWORD_MAX_ATTEMPTS) {
    // Check if user already hit cooldown before (for progressive increase)
    const existingCooldown = await redis.get(cooldownKey);
    if (existingCooldown) {
      // User hit cooldown before, apply longer cooldown
      cooldownDuration = FORGOT_PASSWORD_LONG_COOLDOWN_SEC;
    } else {
      // First time exceeding limit, apply standard cooldown
      cooldownDuration = FORGOT_PASSWORD_COOLDOWN_SEC;
    }
  } else {
    // Within limit, no cooldown needed
    return 0;
  }
  
  await redis.set(cooldownKey, "1", "EX", cooldownDuration);
  return cooldownDuration;
}

/**
 * Check if forgot password request should be allowed (checks both cooldown and attempts)
 * @param email - User email
 * @returns Promise<{ allowed: boolean; remainingCooldown: number; attempts: number }>
 */
export async function checkForgotPasswordAllowed(email: string): Promise<{
  allowed: boolean;
  remainingCooldown: number;
  attempts: number;
}> {
  const remainingCooldown = await checkForgotPasswordCooldown(email);
  const attempts = await getForgotPasswordAttempts(email);
  
  // If in cooldown, not allowed
  if (remainingCooldown > 0) {
    return { allowed: false, remainingCooldown, attempts };
  }
  
  // If attempts exceed limit and not in cooldown, need to set cooldown
  if (attempts >= FORGOT_PASSWORD_MAX_ATTEMPTS) {
    return { allowed: false, remainingCooldown: 0, attempts };
  }
  
  return { allowed: true, remainingCooldown: 0, attempts };
}

/**
 * Clear forgot password cooldown and reset attempts
 * @param email - User email
 * @returns Promise<void>
 */
export async function clearForgotPasswordCooldown(email: string): Promise<void> {
  const cooldownKey = getForgotPasswordCooldownKey(email);
  await redis.del(cooldownKey);
  await resetForgotPasswordAttempts(email);
}

/**
 * Check if reset password attempt is in cooldown
 * @param email - User email
 * @returns Promise<number> - Remaining cooldown seconds, 0 if no cooldown
 */
export async function checkResetPasswordCooldown(email: string): Promise<number> {
  const key = getResetPasswordCooldownKey(email);
  const remaining = await redis.ttl(key);
  return remaining > 0 ? remaining : 0;
}

/**
 * Get reset password attempt count
 * @param email - User email
 * @returns Promise<number> - Current attempt count
 */
async function getResetPasswordAttempts(email: string): Promise<number> {
  const key = getResetPasswordAttemptsKey(email);
  const attempts = await redis.get(key);
  return attempts ? parseInt(attempts, 10) : 0;
}

/**
 * Increment reset password attempt count
 * @param email - User email
 * @returns Promise<number> - New attempt count
 */
async function incrementResetPasswordAttempts(email: string): Promise<number> {
  const key = getResetPasswordAttemptsKey(email);
  const attempts = await redis.incr(key);
  
  // Set expiration on first attempt (24 hours)
  if (attempts === 1) {
    await redis.expire(key, 86400); // 24 hours
  }
  
  return attempts;
}

/**
 * Reset reset password attempt count
 * @param email - User email
 * @returns Promise<void>
 */
async function resetResetPasswordAttempts(email: string): Promise<void> {
  const key = getResetPasswordAttemptsKey(email);
  await redis.del(key);
}

/**
 * Set reset password cooldown with progressive cooldown based on attempts
 * @param email - User email
 * @param isFailedAttempt - Whether this is a failed attempt (true) or successful (false)
 * @returns Promise<number> - Cooldown duration in seconds that was set (0 if no cooldown)
 */
export async function setResetPasswordCooldown(email: string, isFailedAttempt: boolean = true): Promise<number> {
  const cooldownKey = getResetPasswordCooldownKey(email);
  
  // If it's a failed attempt, increment and check
  if (isFailedAttempt) {
    const attempts = await incrementResetPasswordAttempts(email);
    
    // If attempts exceed limit, apply long cooldown
    if (attempts >= RESET_PASSWORD_MAX_ATTEMPTS) {
      await redis.set(cooldownKey, "1", "EX", RESET_PASSWORD_LONG_COOLDOWN_SEC);
      return RESET_PASSWORD_LONG_COOLDOWN_SEC;
    }
    
    // For failed attempts under limit, apply short cooldown
    await redis.set(cooldownKey, "1", "EX", RESET_PASSWORD_COOLDOWN_SEC);
    return RESET_PASSWORD_COOLDOWN_SEC;
  } else {
    // On successful reset, apply short cooldown to prevent rapid successive resets
    await redis.set(cooldownKey, "1", "EX", RESET_PASSWORD_COOLDOWN_SEC);
    return RESET_PASSWORD_COOLDOWN_SEC;
  }
}

/**
 * Check if reset password attempt should be allowed (checks both cooldown and attempts)
 * @param email - User email
 * @returns Promise<{ allowed: boolean; remainingCooldown: number; attempts: number }>
 */
export async function checkResetPasswordAllowed(email: string): Promise<{
  allowed: boolean;
  remainingCooldown: number;
  attempts: number;
}> {
  const remainingCooldown = await checkResetPasswordCooldown(email);
  const attempts = await getResetPasswordAttempts(email);
  
  // If in cooldown, not allowed
  if (remainingCooldown > 0) {
    return { allowed: false, remainingCooldown, attempts };
  }
  
  return { allowed: true, remainingCooldown: 0, attempts };
}

/**
 * Clear reset password cooldown and reset attempts
 * @param email - User email
 * @returns Promise<void>
 */
export async function clearResetPasswordCooldown(email: string): Promise<void> {
  const cooldownKey = getResetPasswordCooldownKey(email);
  await redis.del(cooldownKey);
  await resetResetPasswordAttempts(email);
}

/**
 * Clear all password reset cooldowns for an email
 * Useful after successful password reset
 * @param email - User email
 * @returns Promise<void>
 */
export async function clearAllPasswordResetCooldowns(email: string): Promise<void> {
  await Promise.all([
    clearForgotPasswordCooldown(email),
    clearResetPasswordCooldown(email),
  ]);
}

