import { Router } from "express";
import { forgotPassword, loginUser, logoutUser, registerUser, resetPassword, verifyResetOtp, verifyEmailOtp, resendVerificationOtp, googleMobileAuth, refreshToken, verifyDevice, resendDeviceVerificationOtp, getMyProfile } from "../controllers/auth.controller";
import { authenticate, authenticateDeactivated } from "../middleware";
import {
  enable2FA,
  verify2FASetup,
  disable2FA,
  verify2FALogin,
  get2FAStatus,
  regenerateBackupCodes,
} from "../controllers/twoFactor.controller";
import {
  deactivateAccount,
  deleteAccount,
  deleteProfileImage,
  confirmReactivation,
} from "../controllers/account.controller";
import { getActivity } from "../controllers/activity.controller";
import {
  getSessions,
  revokeSessionById,
  revokeAllSessions,
  getSessionById,
  cleanupSessions,
} from "../controllers/sessions.controller";
import {
  loginLimiter,
  registerLimiter,
  forgotPasswordLimiter,
  otpVerifyLimiter,
  refreshTokenLimiter,
  generalApiLimiter,
} from "../middleware/rateLimiter.middleware";

const router = Router();

// Auth routes - with rate limiting
router.post("/register", registerLimiter, registerUser);
router.post("/login", loginLimiter, loginUser);
router.post("/logout", authenticate, logoutUser);
router.post("/refresh", refreshTokenLimiter, refreshToken);

// Profile
router.get("/myprofile", authenticate, generalApiLimiter, getMyProfile);

// Google OAuth (Mobile App - ID Token Verification)
router.post("/google/mobile", loginLimiter, googleMobileAuth); // Accepts idToken in body, returns tokens in JSON

// Password recovery - with rate limiting
router.post("/forgot-password", forgotPasswordLimiter, forgotPassword);
router.post("/verify-reset-otp", otpVerifyLimiter, verifyResetOtp);
router.post("/reset-password", otpVerifyLimiter, resetPassword);

// Email verification - with rate limiting
router.post("/verify-email-otp", otpVerifyLimiter, verifyEmailOtp);
router.post("/resend-verification-otp", forgotPasswordLimiter, resendVerificationOtp);

// Device verification - with rate limiting
router.post("/verify-device", otpVerifyLimiter, verifyDevice);
router.post("/resend-device-verification-otp", forgotPasswordLimiter, resendDeviceVerificationOtp);

// 2FA routes
router.post("/2fa/verify-login", authenticate, otpVerifyLimiter, verify2FALogin); // Requires authentication from login step
router.get("/2fa/status", authenticate, generalApiLimiter, get2FAStatus);
router.post("/2fa/enable", authenticate, generalApiLimiter, enable2FA);
router.post("/2fa/verify-setup", authenticate, otpVerifyLimiter, verify2FASetup);
router.post("/2fa/disable", authenticate, generalApiLimiter, disable2FA);
router.post("/2fa/regenerate-backup-codes", authenticate, generalApiLimiter, regenerateBackupCodes);

// Account management (Danger Zone)
router.post("/account/deactivate", authenticate, generalApiLimiter, deactivateAccount);
router.post("/account/confirm-reactivation", authenticateDeactivated, generalApiLimiter, confirmReactivation);
router.post("/account/delete", authenticate, generalApiLimiter, deleteAccount);
router.delete("/account/profile-image", authenticate, generalApiLimiter, deleteProfileImage);

// Activity and Sessions
router.get("/activity", authenticate, generalApiLimiter, getActivity);
router.get("/sessions", authenticate, generalApiLimiter, getSessions);
router.delete("/sessions/cleanup", authenticate, generalApiLimiter, cleanupSessions); // Clean up expired sessions
// IMPORTANT: Put /all route BEFORE /:sessionId route to avoid route conflict
router.delete("/sessions/all", authenticate, generalApiLimiter, revokeAllSessions);
router.get("/sessions/:sessionId", authenticate, generalApiLimiter, getSessionById); // Get session details
router.delete("/sessions/:sessionId", authenticate, generalApiLimiter, revokeSessionById);

export default router;
