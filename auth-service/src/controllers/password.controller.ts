import { Request, Response, NextFunction } from "express";
import bcrypt from "bcrypt";
import prisma from "../libs/prisma";
import { BadRequestError, UnauthorizedError, TooManyRequestsError } from "../utils/errors";
import { createAndStoreOtp, verifyOtp, verifyOtpWithoutConsuming } from "../utils/otp";
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
        const { emailOrUsername } = req.body as { emailOrUsername?: string };
        if (!emailOrUsername) throw new BadRequestError("Email or username is required");

        // Find user by email or username
        const user = await prisma.user.findFirst({
            where: {
                OR: [{ email: emailOrUsername }, { username: emailOrUsername }],
            },
        });

        if (!user) {
            throw new BadRequestError("User not found");
        }

        const userEmail = user.email;

        // Check if request is allowed (checks cooldown and attempts)
        const { allowed, remainingCooldown, attempts } = await checkForgotPasswordAllowed(userEmail);
        
        if (!allowed) {
            if (remainingCooldown > 0) {
                const minutes = Math.ceil(remainingCooldown / 60);
                throw new TooManyRequestsError(
                    `Too many password reset requests. Please wait ${minutes} minute${minutes > 1 ? "s" : ""} before trying again.`,
                    { cooldownRemaining: remainingCooldown, retryAfter: remainingCooldown, attempts }
                );
            } else {
                // Attempts exceeded, set cooldown and return error
                const cooldownDuration = await setForgotPasswordCooldown(userEmail);
                const minutes = Math.ceil(cooldownDuration / 60);
                throw new TooManyRequestsError(
                    `Too many password reset requests. Please wait ${minutes} minute${minutes > 1 ? "s" : ""} before trying again.`,
                    { cooldownRemaining: cooldownDuration, retryAfter: cooldownDuration, attempts }
                );
            }
        }

        // Set cooldown before sending OTP (tracks attempts)
        const cooldownDuration = await setForgotPasswordCooldown(userEmail);
        // If cooldown was set (attempts exceeded), return error
        if (cooldownDuration > 0) {
            const minutes = Math.ceil(cooldownDuration / 60);
            throw new TooManyRequestsError(
                `Too many password reset requests. Please wait ${minutes} minute${minutes > 1 ? "s" : ""} before trying again.`,
                { cooldownRemaining: cooldownDuration, retryAfter: cooldownDuration, attempts: attempts + 1 }
            );
        }

        const otp = await createAndStoreOtp(`reset:${userEmail}`);
        // Get user language preference
        const userLanguage = await getUserLanguageByEmail(userEmail);
        // Send OTP via email (non-blocking)
        sendPasswordResetOTP(userEmail, otp, user.name, userLanguage).catch(console.error);
        res.json({ message: "If the email exists, an OTP has been sent.", otp: process.env.NODE_ENV === "production" ? undefined : otp });
    } catch (err) {
        next(err);
    }
}

export const resetPassword = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const { emailOrUsername, otp, newPassword } = req.body as { emailOrUsername?: string; otp?: string; newPassword?: string };
        if (!emailOrUsername || !otp || !newPassword) throw new BadRequestError("Missing required fields");

        // Find user by email or username
        const user = await prisma.user.findFirst({
            where: {
                OR: [{ email: emailOrUsername }, { username: emailOrUsername }],
            },
        });

        if (!user) {
            throw new BadRequestError("User not found");
        }

        const userEmail = user.email;

        // Check if reset attempt is allowed (checks cooldown and attempts)
        const { allowed, remainingCooldown, attempts } = await checkResetPasswordAllowed(userEmail);
        
        if (!allowed) {
            const minutes = Math.ceil(remainingCooldown / 60);
            throw new TooManyRequestsError(
                `Too many reset attempts. Please wait ${minutes} minute${minutes > 1 ? "s" : ""} before trying again.`,
                { cooldownRemaining: remainingCooldown, retryAfter: remainingCooldown, attempts }
            );
        }

        const ok = await verifyOtp(`reset:${userEmail}`, otp);
        if (!ok) {
            // Set cooldown on failed attempt (tracks attempts, applies 30min cooldown after 3 failed attempts)
            const cooldownDuration = await setResetPasswordCooldown(userEmail, true);
            
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
        await setResetPasswordCooldown(userEmail, false);
        
        const hashed = await bcrypt.hash(newPassword, 10);
        await prisma.user.update({ where: { email: userEmail }, data: { password: hashed } });
        
        // Clear all password reset cooldowns and attempts on successful reset
        await clearAllPasswordResetCooldowns(userEmail);
        
        res.json({ message: "Password reset successful" });
    } catch (err) {
        next(err);
    }
}

export const verifyResetOtp = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const { emailOrUsername, otp } = req.body as { emailOrUsername?: string; otp?: string };
        if (!emailOrUsername || !otp) throw new BadRequestError("Missing required fields");

        // Find user by email or username
        const user = await prisma.user.findFirst({
            where: {
                OR: [{ email: emailOrUsername }, { username: emailOrUsername }],
            },
        });

        if (!user) {
            throw new BadRequestError("User not found");
        }

        const userEmail = user.email;

        // Check if verification attempt is allowed (checks cooldown and attempts)
        const { allowed, remainingCooldown, attempts } = await checkResetPasswordAllowed(userEmail);
        
        if (!allowed) {
            const minutes = Math.ceil(remainingCooldown / 60);
            throw new TooManyRequestsError(
                `Too many verification attempts. Please wait ${minutes} minute${minutes > 1 ? "s" : ""} before trying again.`,
                { cooldownRemaining: remainingCooldown, retryAfter: remainingCooldown, attempts }
            );
        }

        // Verify OTP without consuming it
        const ok = await verifyOtpWithoutConsuming(`reset:${userEmail}`, otp);
        if (!ok) {
            // Set cooldown on failed attempt
            const cooldownDuration = await setResetPasswordCooldown(userEmail, true);
            
            if (cooldownDuration >= 1800) {
                const minutes = Math.ceil(cooldownDuration / 60);
                throw new TooManyRequestsError(
                    `Too many failed verification attempts. Please wait ${minutes} minute${minutes > 1 ? "s" : ""} before trying again.`,
                    { cooldownRemaining: cooldownDuration, retryAfter: cooldownDuration, attempts: attempts + 1 }
                );
            }
            
            throw new UnauthorizedError("Invalid or expired OTP");
        }

        res.json({ message: "OTP verified", valid: true });
    } catch (err) {
        next(err);
    }
}
