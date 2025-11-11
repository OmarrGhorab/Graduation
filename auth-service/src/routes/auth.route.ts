import { Router } from "express";
import { forgotPassword, loginUser, logoutUser, registerUser, resetPassword, verifyEmailOtp, resendVerificationOtp, googleAuth, googleCallback, refreshToken } from "../controllers/auth.controller";

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

export default router;
