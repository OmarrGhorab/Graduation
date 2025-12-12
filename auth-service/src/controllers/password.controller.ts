import { Request, Response, NextFunction } from "express";
import bcrypt from "bcrypt";
import prisma from "../libs/prisma";
import { BadRequestError, UnauthorizedError, TooManyRequestsError } from "../utils/errors";
import { createAndStoreOtp, verifyOtp } from "../utils/otp";
import {
  checkForgotPasswordAllowed,
  setForgotPasswordCooldown,
  checkResetPasswordAllowed,
  setResetPasswordCooldown,
  clearAllPasswordResetCooldowns,
} from "../utils/passwordReset";
import { sendPasswordResetOTP } from "../utils/email";
import { getUserLanguageByEmail } from "../utils/userLanguage";

export const forgotPassword = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const { email } = req.body as { email?: string };
        if (!email) throw new BadRequestError("Email is required");

        // Check if request is allowed (checks cooldown and attempts)
        const { allowed, remainingCooldown, attempts } = await checkForgotPasswordAllowed(email);
        
        if (!allowed) {
            if (remainingCooldown > 0) {
                const minutes = Math.ceil(remainingCooldown / 60);
                throw new TooManyRequestsError(
                    `Too many password reset requests. Please wait ${minutes} minute${minutes > 1 ? "s" : ""} before trying again.`,
                    { cooldownRemaining: remainingCooldown, retryAfter: remainingCooldown, attempts }
                );
            } else {
                // Attempts exceeded, set cooldown and return error
                const cooldownDuration = await setForgotPasswordCooldown(email);
                const minutes = Math.ceil(cooldownDuration / 60);
                throw new TooManyRequestsError(
                    `Too many password reset requests. Please wait ${minutes} minute${minutes > 1 ? "s" : ""} before trying again.`,
                    { cooldownRemaining: cooldownDuration, retryAfter: cooldownDuration, attempts }
                );
            }
        }

        const user = await prisma.user.findUnique({ where: { email } });
        if (!user) {
            // Set cooldown even if user doesn't exist to prevent email enumeration
            await setForgotPasswordCooldown(email);
            // do not reveal existence
            return res.json({ message: "If the email exists, an OTP has been sent." });
        }

        // Set cooldown before sending OTP (tracks attempts)
        const cooldownDuration = await setForgotPasswordCooldown(email);
        // If cooldown was set (attempts exceeded), return error
        if (cooldownDuration > 0) {
            const minutes = Math.ceil(cooldownDuration / 60);
            throw new TooManyRequestsError(
                `Too many password reset requests. Please wait ${minutes} minute${minutes > 1 ? "s" : ""} before trying again.`,
                { cooldownRemaining: cooldownDuration, retryAfter: cooldownDuration, attempts: attempts + 1 }
            );
        }

        const otp = await createAndStoreOtp(`reset:${email}`);
        // Get user language preference
        const userLanguage = await getUserLanguageByEmail(email);
        // Send OTP via email (non-blocking)
        sendPasswordResetOTP(email, otp, user?.name, userLanguage).catch(console.error);
        res.json({ message: "If the email exists, an OTP has been sent.", otp: process.env.NODE_ENV === "production" ? undefined : otp });
    } catch (err) {
        next(err);
    }
}

export const resetPassword = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const { email, otp, newPassword } = req.body as { email?: string; otp?: string; newPassword?: string };
        if (!email || !otp || !newPassword) throw new BadRequestError("Missing required fields");

        // Check if reset attempt is allowed (checks cooldown and attempts)
        const { allowed, remainingCooldown, attempts } = await checkResetPasswordAllowed(email);
        
        if (!allowed) {
            const minutes = Math.ceil(remainingCooldown / 60);
            throw new TooManyRequestsError(
                `Too many reset attempts. Please wait ${minutes} minute${minutes > 1 ? "s" : ""} before trying again.`,
                { cooldownRemaining: remainingCooldown, retryAfter: remainingCooldown, attempts }
            );
        }

        const ok = await verifyOtp(`reset:${email}`, otp);
        if (!ok) {
            // Set cooldown on failed attempt (tracks attempts, applies 30min cooldown after 3 failed attempts)
            const cooldownDuration = await setResetPasswordCooldown(email, true);
            
            // If long cooldown was applied (30 min), return appropriate error
            if (cooldownDuration >= 1800) {
                const minutes = Math.ceil(cooldownDuration / 60);
                throw new TooManyRequestsError(
                    `Too many failed reset attempts. Please wait ${minutes} minute${minutes > 1 ? "s" : ""} before trying again.`,
                    { cooldownRemaining: cooldownDuration, retryAfter: cooldownDuration, attempts: attempts + 1 }
                );
            }
            
            throw new UnauthorizedError("Invalid or expired OTP");
        }

        // Set short cooldown on success to prevent rapid successive resets
        await setResetPasswordCooldown(email, false);
        
        const hashed = await bcrypt.hash(newPassword, 10);
        await prisma.user.update({ where: { email }, data: { password: hashed } });
        
        // Clear all password reset cooldowns and attempts on successful reset
        await clearAllPasswordResetCooldowns(email);
        
        res.json({ message: "Password reset successful" });
    } catch (err) {
        next(err);
    }
}
