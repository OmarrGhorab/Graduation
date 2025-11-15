import { Request, Response, NextFunction } from "express";
import bcrypt from "bcrypt";
import prisma from "../libs/prisma";
import { BadRequestError, UnauthorizedError, TooManyRequestsError } from "../utils/errors";
import { createAndStoreOtp, verifyOtp } from "../utils/otp";
import { signAccessToken, signAndStoreRefreshToken, verifyRefreshToken, revokeRefreshTokenByJti, rotateRefreshToken } from "../utils/tokens";
import { setAuthCookies, clearAuthCookies, getRefreshTokenFromRequest } from "../utils/cookies";
import { aj } from "../libs/arcjet";
import { generateUsernameSuggestions, generateUniqueUsername } from "../utils/username";
import { OAuth2Client } from "google-auth-library";
import dotenv from "dotenv";
import {
  checkForgotPasswordAllowed,
  setForgotPasswordCooldown,
  checkResetPasswordAllowed,
  setResetPasswordCooldown,
  clearAllPasswordResetCooldowns,
} from "../utils/passwordReset";
import {
  checkEmailVerificationAllowed,
  setEmailVerificationCooldown,
  clearEmailVerificationCooldown,
  checkResendOtpAllowed,
  setResendOtpCooldown,
} from "../utils/emailVerification";
import { extractDeviceInfo, extractDeviceName } from "../utils/device";


dotenv.config();

export const registerUser = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const { name, username, email, password: rawPassword } = req.body as { name?: string; username?: string; email?: string; password?: unknown };
        if (!name || !username || !email || rawPassword === undefined || rawPassword === null) throw new BadRequestError("Missing required fields");
        const decision = await aj.protect(req, { email });
        console.log("Arcjet decision:", decision);

        if (decision.isDenied()) {
            if (decision.reason.isEmail()) {
            return res.status(400).json({ error: "Invalid email" });
            } else {
            return res.status(403).json({ error: "Forbidden" });
            }
        }
        const password = typeof rawPassword === "string" ? rawPassword : String(rawPassword);
        if (password.length < 8) throw new BadRequestError("Password must be at least 8 characters");

        // Check email first
        const existingEmail = await prisma.user.findUnique({ where: { email } });
        if (existingEmail) throw new BadRequestError("Email already in use");

        // Check username separately to provide suggestions if taken
        const existingUsername = await prisma.user.findUnique({ where: { username } });
        if (existingUsername) {
            const suggestions = await generateUsernameSuggestions(username, 3);
            return res.status(400).json({
                error: "Username already taken",
                suggestions: suggestions.length > 0 ? suggestions : undefined,
            });
        }

        const hashed = await bcrypt.hash(password, 10);
        const user = await prisma.user.create({
            data: { name, username, email, password: hashed, verified: false },
        });

        // issue tokens; store refresh in Redis only (no DB storage)
        const accessToken = signAccessToken({ id: user.id, role: user.role });
        const { token: refreshToken } = await signAndStoreRefreshToken(user.id);
        setAuthCookies(res, accessToken, refreshToken);

        // email verification OTP
        const otp = await createAndStoreOtp(`email:${email}`);
        // TODO: send OTP via email provider

        res.status(201).json({
            user: { id: user.id, name: user.name, email: user.email, verified: user.verified },
            message: "Registration successful. Please Verify your account with the OTP Via Email.",
            // expose OTP in non-production for testing
            otp: process.env.NODE_ENV === "production" ? undefined : otp,
        });
    } catch (err) {
        next(err);
    }
}

export const loginUser = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const { emailOrUsername, password: rawPassword } = req.body as {
            emailOrUsername?: string;
            password?: unknown;
        };

        if (!emailOrUsername || rawPassword === undefined || rawPassword === null) {
            throw new BadRequestError("Missing credentials");
        }

        const password = typeof rawPassword === "string" ? rawPassword : String(rawPassword);

        const user = await prisma.user.findFirst({
            where: {
                OR: [{ email: emailOrUsername }, { username: emailOrUsername }],
            },
            select: {
                id: true,
                name: true,
                username: true,
                email: true,
                password: true,
                verified: true,
                role: true,
                profileImg: true,
                onboardingCompleted: true,
                twoFactorEnabled: true,
                deviceBlocked: true,
                pendingDeviceFingerprint: true,
            },
        });

        if (!user || !user.password) throw new UnauthorizedError("Invalid credentials");

        const ok = await bcrypt.compare(password, user.password);
        if (!ok) throw new UnauthorizedError("Invalid credentials");

        //  Unverified users: block login
        if (!user.verified) {
            return res.status(403).json({
                error: "Account not verified",
                message: "Please verify your account before logging in. Check your email for the verification OTP.",
                verified: false,
                requiresVerification: true,
            });
        }

        // Extract device information
        const deviceInfo = extractDeviceInfo(req, req.body.deviceName);
        const deviceName = extractDeviceName(deviceInfo.userAgent || undefined) || deviceInfo.deviceName || "Unknown Device";

        // Check if device exists for this user
        let existingDevice = await prisma.userDevice.findUnique({
            where: {
                userId_deviceFingerprint: {
                    userId: user.id,
                    deviceFingerprint: deviceInfo.fingerprint,
                },
            },
        });

        // Handle device tracking and blocking
        if (!existingDevice) {
            // New device - check device limit
            // Get user's devices, ordered by login count (most used first)
            const userDevices = await prisma.userDevice.findMany({
                where: { userId: user.id },
                orderBy: { loginCount: "desc" },
            });

            // If user has 2 or more devices, block account and require verification
            if (userDevices.length >= 2) {
                // Block account and store pending device fingerprint
                await prisma.user.update({
                    where: { id: user.id },
                    data: {
                        deviceBlocked: true,
                        pendingDeviceFingerprint: deviceInfo.fingerprint,
                    },
                });

                // Create new device (not trusted)
                existingDevice = await prisma.userDevice.create({
                    data: {
                        userId: user.id,
                        deviceFingerprint: deviceInfo.fingerprint,
                        deviceName: deviceName,
                        userAgent: deviceInfo.userAgent,
                        ipAddress: deviceInfo.ipAddress,
                        platform: deviceInfo.platform,
                        loginCount: 0, // Don't increment yet - device not verified
                        lastLoginAt: null,
                        isTrusted: false,
                    },
                });

                // Create device verification OTP
                const otp = await createAndStoreOtp(`device:${user.id}:${deviceInfo.fingerprint}`);

                // TODO: Send OTP via email or SMS

                return res.status(403).json({
                    error: "New device detected",
                    message: "A new device has been detected. Your account has been blocked until device verification is completed. Please check your email for verification code.",
                    deviceBlocked: true,
                    requiresDeviceVerification: true,
                    deviceFingerprint: deviceInfo.fingerprint,
                    // Expose OTP in non-production for testing
                    otp: process.env.NODE_ENV === "production" ? undefined : otp,
                });
            }

            // User has less than 2 devices, create new device (trusted)
            existingDevice = await prisma.userDevice.create({
                data: {
                    userId: user.id,
                    deviceFingerprint: deviceInfo.fingerprint,
                    deviceName: deviceName,
                    userAgent: deviceInfo.userAgent,
                    ipAddress: deviceInfo.ipAddress,
                    platform: deviceInfo.platform,
                    loginCount: 1,
                    lastLoginAt: new Date(),
                    isTrusted: true, // First 2 devices are trusted by default
                },
            });
        } else {
            // Device exists, update login count and last login
            existingDevice = await prisma.userDevice.update({
                where: { id: existingDevice.id },
                data: {
                    loginCount: { increment: 1 },
                    lastLoginAt: new Date(),
                    // Update IP and user agent in case they changed
                    ipAddress: deviceInfo.ipAddress,
                    userAgent: deviceInfo.userAgent,
                    platform: deviceInfo.platform,
                },
            });

            // If account is blocked for this device and device is not trusted, require verification
            if (user.deviceBlocked && user.pendingDeviceFingerprint === deviceInfo.fingerprint && !existingDevice.isTrusted) {
                const otp = await createAndStoreOtp(`device:${user.id}:${deviceInfo.fingerprint}`);
                return res.status(403).json({
                    error: "Device verification required",
                    message: "This device needs to be verified. Please check your email for verification code.",
                    deviceBlocked: true,
                    requiresDeviceVerification: true,
                    deviceFingerprint: deviceInfo.fingerprint,
                    otp: process.env.NODE_ENV === "production" ? undefined : otp,
                });
            }

            // If account is blocked but this device is trusted, unblock account
            if (user.deviceBlocked && existingDevice.isTrusted) {
                await prisma.user.update({
                    where: { id: user.id },
                    data: {
                        deviceBlocked: false,
                        pendingDeviceFingerprint: null,
                    },
                });
            }
        }

        // If account is blocked and this is not the pending device, block login
        if (user.deviceBlocked && user.pendingDeviceFingerprint !== deviceInfo.fingerprint) {
            return res.status(403).json({
                error: "Account blocked",
                message: "Your account has been blocked due to a new device login. Please verify the pending device first.",
                deviceBlocked: true,
                requiresDeviceVerification: true,
            });
        }

        // Check if 2FA is enabled (do this after device check)
        if (user.twoFactorEnabled) {
            // 2FA is enabled - issue temporary access token for 2FA verification
            const tempAccessToken = signAccessToken({ id: user.id, role: user.role });
            setAuthCookies(res, tempAccessToken);
            
            return res.status(200).json({
                message: "2FA verification required",
                requires2FA: true,
            });
        }

        // Verified but not onboarded: issue token but flag the state
        const accessToken = signAccessToken({ id: user.id, role: user.role });
        const { token: refreshToken } = await signAndStoreRefreshToken(user.id);
        setAuthCookies(res, accessToken, refreshToken);

        await prisma.user.update({
            where: { id: user.id },
            data: { lastLoginAt: new Date() },
        });

        return res.json({
            user: {
                id: user.id,
                name: user.name,
                username: user.username,
                email: user.email,
                verified: user.verified,
                onboardingCompleted: user.onboardingCompleted,
                role: user.role,
                profileImg: user.profileImg,
            },
            requiresOnboarding: !user.onboardingCompleted,
        });
    } catch (err) {
        next(err);
    }
};



export const logoutUser = async (req: Request, res: Response, next: NextFunction) => {
    try {
        // Extract refresh token from request
        const refreshToken = getRefreshTokenFromRequest(req);
        
        if (refreshToken) {
            try {
                const payload = await verifyRefreshToken(refreshToken);
                await revokeRefreshTokenByJti(payload.jti);
            } catch {
                // ignore errors when revoking token during logout
            }
        }
        clearAuthCookies(res);
        res.json({ message: "Logged out" });
    } catch (err) {
        next(err);
    }
}

export const refreshToken = async (req: Request, res: Response, next: NextFunction) => {
    try {
        // Extract refresh token from request
        const refreshToken = getRefreshTokenFromRequest(req);
        
        if (!refreshToken) {
            throw new UnauthorizedError("Refresh token is required");
        }

        // Verify the refresh token
        const payload = await verifyRefreshToken(refreshToken);
        const userId = payload.sub;

        // Get user from database to ensure user still exists and get current role
        const user = await prisma.user.findUnique({
            where: { id: userId },
            select: {
                id: true,
                name: true,
                username: true,
                email: true,
                role: true,
                verified: true,
                onboardingCompleted: true,
                profileImg: true,
            },
        });

        if (!user) {
            // User doesn't exist anymore, revoke the token
            await revokeRefreshTokenByJti(payload.jti);
            throw new UnauthorizedError("User not found");
        }

        // Check if user account is verified
        if (!user.verified) {
            throw new UnauthorizedError("Account not verified");
        }

        // Generate new access token with current user role
        const accessToken = signAccessToken({ id: user.id, role: user.role });

        // Rotate refresh token for security (revoke old, create new)
        const { token: newRefreshToken } = await rotateRefreshToken(payload.jti, user.id);

        // Set new tokens in cookies
        setAuthCookies(res, accessToken, newRefreshToken);

        res.json({
            user: {
                id: user.id,
                name: user.name,
                username: user.username,
                email: user.email,
                verified: user.verified,
                onboardingCompleted: user.onboardingCompleted,
                role: user.role,
                profileImg: user.profileImg,
            },
        });
    } catch (err) {
        // Handle JWT verification errors
        if (err instanceof Error) {
            if (err.name === "JsonWebTokenError" || err.name === "TokenExpiredError" || err.message.includes("Refresh token")) {
                return next(new UnauthorizedError("Invalid or expired refresh token"));
            }
        }
        next(err);
    }
}

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
        // TODO: send OTP via email provider
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
            // TODO: send OTP via email provider
            
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

/**
 * Initiates Google OAuth flow - redirects user to Google
 */
export const googleAuth = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const clientId = process.env.GOOGLE_CLIENT_ID;
        const clientSecret = process.env.GOOGLE_CLIENT_SECRET;
        const redirectUri = process.env.GOOGLE_REDIRECT_URI;

        if (!clientId || !clientSecret || !redirectUri) {
            throw new BadRequestError("Google OAuth credentials not configured");
        }

        const oauth2Client = new OAuth2Client(clientId, clientSecret, redirectUri);

        // Generate the authorization URL
        const authUrl = oauth2Client.generateAuthUrl({
            access_type: "offline",
            scope: [
                "https://www.googleapis.com/auth/userinfo.profile",
                "https://www.googleapis.com/auth/userinfo.email",
            ],
            prompt: "consent", // Force consent to get refresh token
        });

        res.redirect(authUrl);
    } catch (err) {
        next(err);
    }
}

/**
 * Handles Google OAuth callback - creates or logs in user
 */
export const googleCallback = async (req: Request, res: Response, next: NextFunction) => {
    try {
        
        // Handle code parameter - try multiple ways to extract it
        let code: string | undefined;
        
        // Method 1: Direct from req.query (Express default)
        const queryCode = req.query.code;
        if (queryCode) {
            if (Array.isArray(queryCode)) {
                code = typeof queryCode[0] === 'string' ? queryCode[0] : undefined;
            } else if (typeof queryCode === 'string') {
                code = queryCode;
            }
        }
        
        // Method 2: If not found, try parsing from URL directly
        if (!code && req.url) {
            const queryString = req.url.split('?')[1] || '';
            const urlParams = new URLSearchParams(queryString);
            code = urlParams.get('code') || undefined;
        }
        
        if (!code || typeof code !== 'string') {
            console.error("Code parameter missing or invalid:", { 
                code, 
                type: typeof code, 
                query: req.query,
                hasCodeInQuery: !!req.query.code,
                url: req.url
            });
            throw new BadRequestError("Authorization code not provided");
        }
        

        const clientId = process.env.GOOGLE_CLIENT_ID;
        const clientSecret = process.env.GOOGLE_CLIENT_SECRET;
        const redirectUri = process.env.GOOGLE_REDIRECT_URI;

        if (!clientId || !clientSecret || !redirectUri) {
            throw new BadRequestError("Google OAuth credentials not configured");
        }

        const oauth2Client = new OAuth2Client(clientId, clientSecret, redirectUri);

        // Exchange code for tokens
        const { tokens } = await oauth2Client.getToken(code);
        oauth2Client.setCredentials(tokens);

        // Get user info from Google
        const ticket = await oauth2Client.verifyIdToken({
            idToken: tokens.id_token!,
            audience: clientId,
        });

        const payload = ticket.getPayload();
        if (!payload) throw new BadRequestError("Failed to get user info from Google");

        const googleId = payload.sub;
        const email = payload.email;
        const name = payload.name || "";
        const picture = payload.picture;

        if (!email) throw new BadRequestError("Email not provided by Google");

        // Check if user exists by email
        let user = await prisma.user.findUnique({
            where: { email },
            include: { providers: true },
        });

        // If user doesn't exist, create new user
        if (!user) {
            // Generate unique username from name or email
            const baseUsername = name || email.split("@")[0];
            const username = await generateUniqueUsername(baseUsername);

            user = await prisma.user.create({
                data: {
                    name,
                    username,
                    email,
                    profileImg: picture || undefined,
                    verified: true, // Google email is already verified
                    providers: {
                        create: {
                            provider: "GOOGLE",
                            providerId: googleId,
                            accessToken: tokens.access_token || undefined,
                            refreshToken: tokens.refresh_token || undefined,
                        },
                    },
                },
                include: { providers: true },
            });
        } else {
            // User exists - check if Google provider is linked
            const googleProvider = user.providers.find((p) => p.provider === "GOOGLE");

            if (googleProvider) {
                // Update existing provider tokens
                await prisma.authProvider.update({
                    where: { id: googleProvider.id },
                    data: {
                        accessToken: tokens.access_token || undefined,
                        refreshToken: tokens.refresh_token || undefined,
                    },
                });
            } else {
                // Link Google provider to existing user
                await prisma.authProvider.create({
                    data: {
                        provider: "GOOGLE",
                        providerId: googleId,
                        userId: user.id,
                        accessToken: tokens.access_token || undefined,
                        refreshToken: tokens.refresh_token || undefined,
                    },
                });
            }

            // Update user profile image if available
            if (picture && !user.profileImg) {
                await prisma.user.update({
                    where: { id: user.id },
                    data: { profileImg: picture },
                });
            }

            // Update last login
            await prisma.user.update({
                where: { id: user.id },
                data: { lastLoginAt: new Date(), verified: true },
            });
        }

        // Check if user account is verified (should always be true for Google users, but check for safety)
        if (!user.verified) {
            const frontendUrl = process.env.FRONTEND_URL || "http://localhost:3000";
            return res.redirect(`${frontendUrl}/auth/google/callback?error=verification_required&message=Please verify your account`);
        }

        // Check if onboarding is completed
        if (!user.onboardingCompleted) {
            const frontendUrl = process.env.FRONTEND_URL || "http://localhost:3000";
            // Issue tokens but redirect to onboarding
            const accessToken = signAccessToken({ id: user.id, role: user.role });
            const { token: refreshToken } = await signAndStoreRefreshToken(user.id);
            setAuthCookies(res, accessToken, refreshToken);
            return res.redirect(`${frontendUrl}/auth/google/callback?success=true&token=${accessToken}&requiresOnboarding=true`);
        }

        // Issue tokens
        const accessToken = signAccessToken({ id: user.id, role: user.role });
        const { token: refreshToken } = await signAndStoreRefreshToken(user.id);
        setAuthCookies(res, accessToken, refreshToken);

        // Redirect to frontend with success (you can customize this redirect URL)
        const frontendUrl = process.env.FRONTEND_URL || "http://localhost:3000";
        res.redirect(`${frontendUrl}/auth/google/callback?success=true&token=${accessToken}`);
    } catch (err) {
        next(err);
    }
}

/**
 * Verify device with OTP
 * This endpoint is called when a new device is detected and needs verification
 */
export const verifyDevice = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const { emailOrUsername, deviceFingerprint, otp } = req.body as {
            emailOrUsername?: string;
            deviceFingerprint?: string;
            otp?: string;
        };

        if (!emailOrUsername || !deviceFingerprint || !otp) {
            throw new BadRequestError("Email/username, device fingerprint, and OTP are required");
        }

        // Find user
        const user = await prisma.user.findFirst({
            where: {
                OR: [{ email: emailOrUsername }, { username: emailOrUsername }],
            },
        });

        if (!user) {
            throw new UnauthorizedError("Invalid credentials");
        }

        // Verify OTP
        const otpKey = `device:${user.id}:${deviceFingerprint}`;
        const isValid = await verifyOtp(otpKey, otp);

        if (!isValid) {
            throw new UnauthorizedError("Invalid or expired OTP");
        }

        // Find device
        const device = await prisma.userDevice.findUnique({
            where: {
                userId_deviceFingerprint: {
                    userId: user.id,
                    deviceFingerprint: deviceFingerprint,
                },
            },
        });

        if (!device) {
            throw new BadRequestError("Device not found");
        }

        // Verify that this is the pending device
        if (user.deviceBlocked && user.pendingDeviceFingerprint !== deviceFingerprint) {
            throw new BadRequestError("This device is not the pending device for verification");
        }

        // Mark device as trusted and update login count
        await prisma.userDevice.update({
            where: { id: device.id },
            data: {
                isTrusted: true,
                loginCount: { increment: 1 },
                lastLoginAt: new Date(),
            },
        });

        // Unblock account
        await prisma.user.update({
            where: { id: user.id },
            data: {
                deviceBlocked: false,
                pendingDeviceFingerprint: null,
            },
        });

        // Check if 2FA is enabled
        if (user.twoFactorEnabled) {
            return res.status(200).json({
                message: "Device verified successfully. 2FA verification required.",
                deviceVerified: true,
                requires2FA: true,
                emailOrUsername: emailOrUsername,
            });
        }

        // Issue tokens
        const accessToken = signAccessToken({ id: user.id, role: user.role });
        const { token: refreshToken } = await signAndStoreRefreshToken(user.id);
        setAuthCookies(res, accessToken, refreshToken);

        await prisma.user.update({
            where: { id: user.id },
            data: { lastLoginAt: new Date() },
        });

        return res.json({
            message: "Device verified successfully",
            deviceVerified: true,
            user: {
                id: user.id,
                name: user.name,
                username: user.username,
                email: user.email,
                verified: user.verified,
                onboardingCompleted: user.onboardingCompleted,
                role: user.role,
                profileImg: user.profileImg,
            },
            requiresOnboarding: !user.onboardingCompleted,
        });
    } catch (err) {
        next(err);
    }
};

/**
 * Resend device verification OTP
 */
export const resendDeviceVerificationOtp = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const { emailOrUsername, deviceFingerprint } = req.body as {
            emailOrUsername?: string;
            deviceFingerprint?: string;
        };

        if (!emailOrUsername || !deviceFingerprint) {
            throw new BadRequestError("Email/username and device fingerprint are required");
        }

        // Find user
        const user = await prisma.user.findFirst({
            where: {
                OR: [{ email: emailOrUsername }, { username: emailOrUsername }],
            },
        });

        if (!user) {
            // Don't reveal if user exists
            return res.json({
                message: "If the email/username exists and device verification is required, an OTP has been sent.",
            });
        }

        // Check if device exists and is pending verification
        const device = await prisma.userDevice.findUnique({
            where: {
                userId_deviceFingerprint: {
                    userId: user.id,
                    deviceFingerprint: deviceFingerprint,
                },
            },
        });

        if (!device || user.pendingDeviceFingerprint !== deviceFingerprint) {
            return res.status(400).json({
                error: "Device verification not required",
                message: "This device does not require verification.",
            });
        }

        // Generate new OTP
        const otp = await createAndStoreOtp(`device:${user.id}:${deviceFingerprint}`);

        // TODO: Send OTP via email or SMS

        return res.json({
            message: "Device verification OTP has been sent.",
            // Expose OTP in non-production for testing
            otp: process.env.NODE_ENV === "production" ? undefined : otp,
        });
    } catch (err) {
        next(err);
    }
};
