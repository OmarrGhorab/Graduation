import { Request, Response, NextFunction } from "express";
import prisma from "../libs/prisma";
import { BadRequestError, UnauthorizedError, TooManyRequestsError } from "../utils/errors";
import { createAndStoreOtp, verifyOtp } from "../utils/otp";
import {
  checkEmailVerificationAllowed,
  setEmailVerificationCooldown,
  clearEmailVerificationCooldown,
  checkResendOtpAllowed,
  setResendOtpCooldown,
} from "../utils/emailVerification";
import { sendVerificationOTP } from "../utils/email";
import { getUserLanguage } from "../utils/userLanguage";

export const resendVerificationOtp = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const { email } = req.body as { email?: string };
        if (!email) throw new BadRequestError("Email is required");

        // Check if resend request is allowed (checks cooldown and attempts)
        const { allowed, remainingCooldown, attempts } = await checkResendOtpAllowed(email);
        
        if (!allowed) {
            // Cooldown is already set by checkResendOtpAllowed if attempts exceeded
            const minutes = Math.ceil(remainingCooldown / 60);
            const seconds = remainingCooldown % 60;
            const timeMessage = minutes > 0 
                ? `${minutes} minute${minutes > 1 ? "s" : ""}${seconds > 0 ? ` and ${seconds} second${seconds > 1 ? "s" : ""}` : ""}`
                : `${seconds} second${seconds > 1 ? "s" : ""}`;
            
            throw new TooManyRequestsError(
                `Too many resend requests. Please wait ${timeMessage} before trying again.`,
                { cooldownRemaining: remainingCooldown, retryAfter: remainingCooldown, attempts }
            );
        }

        // Check if user exists
        const user = await prisma.user.findUnique({ where: { email } });
        
        // Check if user is already verified (only if user exists)
        if (user && user.verified) {
            // Set cooldown to prevent enumeration attacks (same response time)
            await setResendOtpCooldown(email);
            return res.status(400).json({ 
                error: "Email already verified",
                message: "This email address has already been verified.",
            });
        }

        // Set cooldown before sending OTP (tracks attempts and prevents rapid resends)
        // This also increments the attempt counter
        await setResendOtpCooldown(email);

        // Generate and store new OTP only if user exists and is not verified
        if (user && !user.verified) {
            const otp = await createAndStoreOtp(`email:${email}`);
            // Get user language preference
            const userLanguage = await getUserLanguage(user.id);
            // Send OTP via email (non-blocking)
            sendVerificationOTP(email, otp, user.name, userLanguage).catch(console.error);
            
            res.json({ 
                message: "Verification OTP has been sent to your email.",
                // Expose OTP in non-production for testing
                otp: process.env.NODE_ENV === "production" ? undefined : otp,
            });
        } else {
            // User doesn't exist
            // Don't reveal if email exists to prevent enumeration
            // Cooldown is already set to prevent abuse
            res.json({ 
                message: "If the email exists and is not verified, a verification OTP has been sent.",
            });
        }
    } catch (err) {
        next(err);
    }
}

export const verifyEmailOtp = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const { email, otp } = req.body as { email?: string; otp?: string };
        if (!email || !otp) throw new BadRequestError("Email and OTP are required");

        // Check if verification attempt is allowed (checks cooldown and attempts)
        const { allowed, remainingCooldown, attempts } = await checkEmailVerificationAllowed(email);
        
        if (!allowed) {
            if (remainingCooldown > 0) {
                const minutes = Math.ceil(remainingCooldown / 60);
                throw new TooManyRequestsError(
                    `Too many verification attempts. Please wait ${minutes} minute${minutes > 1 ? "s" : ""} before trying again.`,
                    { cooldownRemaining: remainingCooldown, retryAfter: remainingCooldown, attempts }
                );
            } else {
                // Attempts exceeded, set cooldown and return error
                const cooldownDuration = await setEmailVerificationCooldown(email);
                const minutes = Math.ceil(cooldownDuration / 60);
                throw new TooManyRequestsError(
                    `Too many verification attempts. Please wait ${minutes} minute${minutes > 1 ? "s" : ""} before trying again.`,
                    { cooldownRemaining: cooldownDuration, retryAfter: cooldownDuration, attempts }
                );
            }
        }

        const ok = await verifyOtp(`email:${email}`, otp);
        if (!ok) {
            // Set cooldown on failed attempt (tracks attempts, applies progressive cooldown)
            const cooldownDuration = await setEmailVerificationCooldown(email);
            
            // If cooldown was applied, return appropriate error
            if (cooldownDuration > 0) {
                const minutes = Math.ceil(cooldownDuration / 60);
                throw new TooManyRequestsError(
                    `Too many failed verification attempts. Please wait ${minutes} minute${minutes > 1 ? "s" : ""} before trying again.`,
                    { cooldownRemaining: cooldownDuration, retryAfter: cooldownDuration, attempts: attempts + 1 }
                );
            }
            
            throw new UnauthorizedError("Invalid or expired OTP");
        }

        // Clear cooldown and attempts on successful verification
        await clearEmailVerificationCooldown(email);

        const user = await prisma.user.update({ where: { email }, data: { verified: true } });
        res.json({ message: "Email verified", user: { id: user.id, email: user.email, verified: user.verified } });
    } catch (err) {
        next(err);
    }
}
