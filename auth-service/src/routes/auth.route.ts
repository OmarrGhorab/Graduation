import { Router, RequestHandler } from "express";
import { register } from "../controllers/auth.controller";

const router = Router();

const notImplemented: RequestHandler = (req, res) => {
  res.status(501).json({ message: "Not implemented" });
};

router.post("/register", register as RequestHandler);
router.post("/login", notImplemented);
router.post("/logout", notImplemented);
router.post("/forgot-password", notImplemented);
router.post("/reset-password", notImplemented);
router.post("/verify-email", notImplemented);
router.post("/verify-email-otp", notImplemented);
router.post("/send-email-otp", notImplemented);
router.post("/send-email-verification", notImplemented);
router.post("/send-email-reset-password", notImplemented);
router.post("/send-email-verification-otp", notImplemented);
router.post("/send-email-reset-password-otp", notImplemented);

export default router;


