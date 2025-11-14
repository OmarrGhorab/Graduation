import { Router } from "express";
import { forgotPassword, loginUser, logoutUser, registerUser, resetPassword, verifyEmailOtp, resendVerificationOtp, googleAuth, googleCallback, refreshToken } from "../controllers/auth.controller";
import { authenticate } from "../middleware";
import {
  enable2FA,
  verify2FASetup,
  disable2FA,
  verify2FALogin,
  get2FAStatus,
  regenerateBackupCodes,
} from "../controllers/twoFactor.controller";

const router = Router();

// Auth routes
router.post("/register", registerUser);
router.post("/login", loginUser);
router.post("/logout", logoutUser);
router.post("/refresh", refreshToken);

// Google OAuth
router.get("/google", googleAuth);
router.get("/google/callback", googleCallback);

// Password recovery
router.post("/forgot-password", forgotPassword);
router.post("/reset-password", resetPassword);

// Email verification
router.post("/verify-email-otp", verifyEmailOtp);
router.post("/resend-verification-otp", resendVerificationOtp);

// 2FA routes
router.post("/2fa/verify-login", verify2FALogin); // Public route for 2FA verification during login
router.get("/2fa/status", authenticate, get2FAStatus);
router.post("/2fa/enable", authenticate, enable2FA);
router.post("/2fa/verify-setup", authenticate, verify2FASetup);
router.post("/2fa/disable", authenticate, disable2FA);
router.post("/2fa/regenerate-backup-codes", authenticate, regenerateBackupCodes);

export default router;
