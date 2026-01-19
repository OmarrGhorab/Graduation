import rateLimit from "express-rate-limit";
import { Request, Response } from "express";

/**
 * Rate limiter configurations for different auth endpoints
 * OPTIMIZED: Prevents brute force attacks and abuse
 */

// Standard error response format
const rateLimitResponse = (message: string) => ({
  error: "Too Many Requests",
  message,
  retryAfter: "Please try again later",
});

/**
 * Strict rate limiter for login attempts
 * 5 attempts per 15 minutes per IP
 */
export const loginLimiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 5, // 5 attempts per window
  message: rateLimitResponse("Too many login attempts. Please try again in 15 minutes."),
  standardHeaders: true, // Return rate limit info in headers
  legacyHeaders: false,
  skip: (req: Request) => process.env.NODE_ENV !== "production",
});

/**
 * Rate limiter for registration
 * 3 registrations per hour per IP
 */
export const registerLimiter = rateLimit({
  windowMs: 60 * 60 * 1000, // 1 hour
  max: 3, // 3 registrations per hour
  message: rateLimitResponse("Too many registration attempts. Please try again in an hour."),
  standardHeaders: true,
  legacyHeaders: false,
  skip: (req: Request) => process.env.NODE_ENV !== "production",
});

/**
 * Rate limiter for password reset requests
 * 3 requests per 15 minutes per IP
 */
export const forgotPasswordLimiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 3,
  message: rateLimitResponse("Too many password reset requests. Please try again in 15 minutes."),
  standardHeaders: true,
  legacyHeaders: false,
  skip: (req: Request) => process.env.NODE_ENV !== "production",
});

/**
 * Rate limiter for OTP verification attempts
 * 10 attempts per 15 minutes per IP
 */
export const otpVerifyLimiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 10,
  message: rateLimitResponse("Too many verification attempts. Please try again in 15 minutes."),
  standardHeaders: true,
  legacyHeaders: false,
  skip: (req: Request) => process.env.NODE_ENV !== "production",
});

/**
 * Rate limiter for token refresh
 * 30 refreshes per minute per IP (allows normal usage but prevents abuse)
 */
export const refreshTokenLimiter = rateLimit({
  windowMs: 60 * 1000, // 1 minute
  max: 30,
  message: rateLimitResponse("Too many token refresh requests. Please try again shortly."),
  standardHeaders: true,
  legacyHeaders: false,
  skip: (req: Request) => process.env.NODE_ENV !== "production",
});

/**
 * General API rate limiter for authenticated endpoints
 * 100 requests per minute per IP
 */
export const generalApiLimiter = rateLimit({
  windowMs: 60 * 1000, // 1 minute
  max: 100,
  message: rateLimitResponse("Too many requests. Please slow down."),
  standardHeaders: true,
  legacyHeaders: false,
  skip: (req: Request) => process.env.NODE_ENV !== "production",
});
