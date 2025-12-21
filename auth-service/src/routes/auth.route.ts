import { Router } from "express";
import { forgotPassword, loginUser, logoutUser, registerUser, resetPassword, verifyEmailOtp, resendVerificationOtp, googleMobileAuth, refreshToken, verifyDevice, resendDeviceVerificationOtp } from "../controllers/auth.controller";
import { getMyProfile } from "../controllers/auth.core.controller";
import { authenticate } from "../middleware";
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
} from "../controllers/account.controller";
import { getActivity } from "../controllers/activity.controller";
import {
  getSessions,
  revokeSessionById,
  revokeAllSessions,
} from "../controllers/sessions.controller";

const router = Router();

// Auth routes
router.post("/register", registerUser);
router.post("/login", loginUser);
router.post("/logout", logoutUser);
router.post("/refresh", refreshToken);

// Profile
router.get("/myprofile", authenticate, getMyProfile);

// Google OAuth (Mobile App - ID Token Verification)
router.post("/google/mobile", googleMobileAuth); // Accepts idToken in body, returns tokens in JSON

// Password recovery
router.post("/forgot-password", forgotPassword);
router.post("/reset-password", resetPassword);

// Email verification
router.post("/verify-email-otp", verifyEmailOtp);
router.post("/resend-verification-otp", resendVerificationOtp);

// Device verification
router.post("/verify-device", verifyDevice);
router.post("/resend-device-verification-otp", resendDeviceVerificationOtp);

// 2FA routes
router.post("/2fa/verify-login", authenticate, verify2FALogin); // Requires authentication from login step
router.get("/2fa/status", authenticate, get2FAStatus);
router.post("/2fa/enable", authenticate, enable2FA);
router.post("/2fa/verify-setup", authenticate, verify2FASetup);
router.post("/2fa/disable", authenticate, disable2FA);
router.post("/2fa/regenerate-backup-codes", authenticate, regenerateBackupCodes);

// Account management (Danger Zone)
router.post("/account/deactivate", authenticate, deactivateAccount);
router.post("/account/delete", authenticate, deleteAccount);
router.delete("/account/profile-image", authenticate, deleteProfileImage);

// Activity and Sessions
router.get("/activity", authenticate, getActivity);
router.get("/sessions", authenticate, getSessions);
// IMPORTANT: Put /all route BEFORE /:sessionId route to avoid route conflict
router.delete("/sessions/all", authenticate, revokeAllSessions);
router.delete("/sessions/:sessionId", authenticate, revokeSessionById);

export default router;
