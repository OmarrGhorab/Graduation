import { Request, Response, NextFunction } from "express";
import prisma from "../libs/prisma";
import { BadRequestError } from "../utils/errors";
import { signAccessToken, signAndStoreRefreshToken } from "../utils/tokens";
import { generateUniqueUsername } from "../utils/username";
import { extractDeviceInfo, extractDeviceName } from "../utils/device";
import { createSession, getSessionDeviceInfo } from "../utils/sessions";
import { createAndStoreOtp } from "../utils/otp";
import { sendDeviceVerificationOTP } from "../utils/email";
import { getUserLanguage } from "../utils/userLanguage";
import { OAuth2Client } from "google-auth-library";
import dotenv from "dotenv";

dotenv.config();

/**
 * Handles Google Sign-In for mobile apps using ID token verification
 * Accepts Google ID token from mobile app and verifies it directly
 */
export const googleMobileAuth = async (req: Request, res: Response, next: NextFunction) => {
    try {
        // Extract ID token from request body
        const idToken = req.body?.idToken;
        
        if (!idToken || typeof idToken !== 'string') {
            throw new BadRequestError("ID token is required in request body");
        }

        const clientId = process.env.GOOGLE_CLIENT_ID;

        if (!clientId) {
            throw new BadRequestError("Google OAuth credentials not configured");
        }

        // Create OAuth2Client with only clientId (no secret or redirectUri needed for ID token verification)
        const client = new OAuth2Client(clientId);

        // Verify the ID token
        const ticket = await client.verifyIdToken({
            idToken: idToken,
            audience: clientId,
        });

        const payload = ticket.getPayload();
        if (!payload) {
            throw new BadRequestError("Failed to verify ID token");
        }

        // Extract user information from verified token
        const googleId = payload.sub;
        const email = payload.email;
        const name = payload.name || "";
        const picture = payload.picture;

        if (!email) {
            throw new BadRequestError("Email not provided by Google");
        }

        // Security: Verify that Google has verified this email
        if (!payload.email_verified) {
            throw new BadRequestError("Google email is not verified");
        }

        // Check if user exists by email
        let user = await prisma.user.findUnique({
            where: { email },
            select: {
                id: true,
                name: true,
                username: true,
                email: true,
                profileImg: true,
                role: true,
                verified: true,
                onboardingCompleted: true,
                twoFactorEnabled: true,
                providers: true,
                isActive: true,
                deletedAt: true,
            },
        });

        // If user doesn't exist, create new user
        if (!user) {
            // Generate unique username from name or email
            const baseUsername = name || email.split("@")[0];
            const username = await generateUniqueUsername(baseUsername);

            // Create new user - automatically verified since Google email is verified
            user = await prisma.user.create({
                data: {
                    name,
                    username,
                    email,
                    profileImg: picture || undefined,
                    verified: true, // Google OAuth users are automatically verified (no email verification needed)
                    providers: {
                        create: {
                            provider: "GOOGLE",
                            providerId: googleId,
                            // Do NOT store Google access/refresh tokens for mobile users
                            accessToken: null,
                            refreshToken: null,
                        },
                    },
                },
                include: { providers: true },
            });
        } else {
            // Check if account is deleted
            if (user.deletedAt) {
                throw new BadRequestError("Account has been deleted");
            }

            // Check if account is deactivated - require confirmation to reactivate
            if (!user.isActive) {
                // Issue temporary token for reactivation confirmation (short-lived)
                const { token: tempToken } = signAccessToken({ 
                    id: user.id, 
                    role: user.role
                });

                return res.status(403).json({
                    success: false,
                    error: "Account deactivated",
                    message: "Your account is deactivated. Would you like to reactivate it?",
                    accountDeactivated: true,
                    requiresReactivation: true,
                    tempToken: tempToken,
                });
            }

            // User exists - check if Google provider is linked
            const googleProvider = user.providers.find((p) => p.provider === "GOOGLE");

            // Security: Prevent account hijacking via email reuse with different Google account
            if (googleProvider && googleProvider.providerId !== googleId) {
                throw new BadRequestError("Google account mismatch");
            }

            if (googleProvider) {
                // Update existing provider (do NOT store Google tokens for mobile)
                await prisma.authProvider.update({
                    where: { id: googleProvider.id },
                    data: {
                        // Do NOT store Google access/refresh tokens for mobile users
                        accessToken: null,
                        refreshToken: null,
                    },
                });
            } else {
                // Link Google provider to existing user (first time linking Google)
                await prisma.authProvider.create({
                    data: {
                        provider: "GOOGLE",
                        providerId: googleId,
                        userId: user.id,
                        // Do NOT store Google access/refresh tokens for mobile users
                        accessToken: null,
                        refreshToken: null,
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

            // Automatically verify user when signing in with Google OAuth
            // This ensures users who sign up or log in with Google are always verified
            // No need for separate email verification step
            await prisma.user.update({
                where: { id: user.id },
                data: { verified: true },
            });
            // Keep local user state in sync for subsequent checks
            user.verified = true;
        }

        // Check if user account is verified (should always be true for Google users, but check for safety)
        if (!user.verified) {
            return res.status(400).json({
                success: false,
                error: "verification_required",
                message: "Please verify your account",
            });
        }

        // Extract device info
        const deviceInfo = extractDeviceInfo(req);
        const sessionDeviceInfo = await getSessionDeviceInfo(req);
        const deviceName = sessionDeviceInfo.deviceName || extractDeviceName(deviceInfo.userAgent);
        
        // Try to find existing device
        let existingDevice = await prisma.userDevice.findUnique({
            where: {
                userId_deviceFingerprint: {
                    userId: user.id,
                    deviceFingerprint: deviceInfo.fingerprint,
                },
            },
        });

        // Handle device tracking and blocking for new devices
        if (!existingDevice) {
            // New device - check device limit
            const userDevices = await prisma.userDevice.findMany({
                where: { userId: user.id },
                orderBy: { lastLoginAt: "desc" },
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
                        ipAddress: sessionDeviceInfo.ipAddress,
                        platform: sessionDeviceInfo.platform as any || deviceInfo.platform,
                        lastLoginAt: null,
                        isTrusted: false,
                    },
                });

                // Create device verification OTP
                const otp = await createAndStoreOtp(`device:${user.id}:${deviceInfo.fingerprint}`);
                const userLanguage = await getUserLanguage(user.id);
                sendDeviceVerificationOTP(user.email, otp, user.name, userLanguage).catch(console.error);

                return res.status(403).json({
                    success: false,
                    error: "New device detected",
                    message: "A new device has been detected. Your account has been blocked until device verification is completed. Please check your email for verification code.",
                    deviceBlocked: true,
                    requiresDeviceVerification: true,
                    deviceFingerprint: deviceInfo.fingerprint,
                    emailOrUsername: user.email,
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
                    ipAddress: sessionDeviceInfo.ipAddress,
                    platform: sessionDeviceInfo.platform as any || deviceInfo.platform,
                    lastLoginAt: new Date(),
                    isTrusted: true,
                },
            });
        } else {
            // Device exists, update last login and info
            existingDevice = await prisma.userDevice.update({
                where: { id: existingDevice.id },
                data: {
                    lastLoginAt: new Date(),
                    ipAddress: sessionDeviceInfo.ipAddress,
                    userAgent: sessionDeviceInfo.userAgent,
                    platform: sessionDeviceInfo.platform as any || deviceInfo.platform,
                },
            });

            // If account is blocked for this device and device is not trusted, require verification
            const currentUser = await prisma.user.findUnique({
                where: { id: user.id },
                select: { deviceBlocked: true, pendingDeviceFingerprint: true },
            });

            if (currentUser?.deviceBlocked && currentUser.pendingDeviceFingerprint === deviceInfo.fingerprint && !existingDevice.isTrusted) {
                const otp = await createAndStoreOtp(`device:${user.id}:${deviceInfo.fingerprint}`);
                const userLanguage = await getUserLanguage(user.id);
                sendDeviceVerificationOTP(user.email, otp, user.name, userLanguage).catch(console.error);

                return res.status(403).json({
                    success: false,
                    error: "Device verification required",
                    message: "This device needs to be verified. Please check your email for verification code.",
                    deviceBlocked: true,
                    requiresDeviceVerification: true,
                    deviceFingerprint: deviceInfo.fingerprint,
                    emailOrUsername: user.email,
                    otp: process.env.NODE_ENV === "production" ? undefined : otp,
                });
            }

            // If account is blocked but this device is trusted, unblock account
            if (currentUser?.deviceBlocked && existingDevice.isTrusted) {
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
        const finalUserState = await prisma.user.findUnique({
            where: { id: user.id },
            select: { deviceBlocked: true, pendingDeviceFingerprint: true },
        });

        if (finalUserState?.deviceBlocked && finalUserState.pendingDeviceFingerprint !== deviceInfo.fingerprint) {
            return res.status(403).json({
                success: false,
                error: "Account blocked",
                message: "Your account has been blocked due to a new device login. Please verify the pending device first.",
                deviceBlocked: true,
                requiresDeviceVerification: true,
            });
        }

        // Issue tokens
        const { token: accessToken, jti: accessJti } = signAccessToken({ id: user.id, role: user.role });
        const { token: refreshToken, jti: refreshJti } = await signAndStoreRefreshToken(user.id);
        
        // Create session record in database
        const expiresAt = new Date();
        expiresAt.setSeconds(expiresAt.getSeconds() + parseInt(process.env.ACCESS_TOKEN_TTL_SEC || "900", 10));
        const refreshExpiresAt = new Date();
        refreshExpiresAt.setSeconds(refreshExpiresAt.getSeconds() + parseInt(process.env.REFRESH_TOKEN_TTL_SEC || "2592000", 10));
        
        const deviceId = existingDevice.id;
        
        // Create session
        await createSession({
            userId: user.id,
            deviceId: deviceId,
            sessionToken: accessJti,
            refreshTokenJti: refreshJti,
            ipAddress: sessionDeviceInfo.ipAddress,
            userAgent: sessionDeviceInfo.userAgent,
            location: sessionDeviceInfo.location,
            expiresAt: expiresAt,
            refreshExpiresAt: refreshExpiresAt,
        });
        
        // Update last login
        await prisma.user.update({
            where: { id: user.id },
            data: { lastLoginAt: new Date() },
        });

        // Check if user has a password set (OAuth-only users won't have one)
        const userWithPassword = await prisma.user.findUnique({
            where: { id: user.id },
            select: { password: true },
        });
        const hasPassword = !!userWithPassword?.password;

        // Return JSON response for mobile apps
        res.json({
            success: true,
            accessToken,
            refreshToken,
            user: {
                id: user.id,
                email: user.email,
                name: user.name,
                username: user.username,
                profileImg: user.profileImg,
                role: user.role,
                verified: user.verified,
                onboardingCompleted: user.onboardingCompleted,
                twoFactorEnabled: user.twoFactorEnabled,
                hasPassword: hasPassword,
                isActive: user.isActive,
            },
            requiresOnboarding: !user.onboardingCompleted,
        });
    } catch (err) {
        next(err);
    }
}
