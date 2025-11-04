import { Router, RequestHandler } from "express";
import { forgotPassword, loginUser, logoutUser, registerUser, resetPassword, verifyEmailOtp, googleAuth, googleCallback } from "../controllers/auth.controller";

const router = Router();

// Auth routes
router.post("/register", registerUser);
router.post("/login", loginUser);
router.post("/logout", logoutUser);

// Google OAuth
router.get("/google", googleAuth);
router.get("/google/callback", googleCallback);

// Password recovery
router.post("/forgot-password", forgotPassword);
router.post("/reset-password", resetPassword);

// Email verification
router.post("/verify-email-otp", verifyEmailOtp);

export default router;
