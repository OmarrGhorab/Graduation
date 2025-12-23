import { Request, Response, NextFunction } from "express";
import bcrypt from "bcrypt";
import prisma from "../libs/prisma";
import { BadRequestError, UnauthorizedError } from "../utils/errors";
import { signAccessToken, signAndStoreRefreshToken, verifyRefreshToken, revokeRefreshTokenByJti, rotateRefreshToken, verifyAccessToken } from "../utils/tokens";
import { getRefreshTokenFromRequest, getAccessTokenFromRequest } from "../utils/cookies";
import { aj } from "../libs/arcjet";
import { generateUsernameSuggestions } from "../utils/username";
import { createAndStoreOtp } from "../utils/otp";
import { extractDeviceInfo, extractDeviceName } from "../utils/device";
import { createSession, getSessionDeviceInfo, revokeSession } from "../utils/sessions";
import { sendVerificationOTP, sendDeviceVerificationOTP } from "../utils/email";
import { getUserLanguage } from "../utils/userLanguage";

export const registerUser = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const { name, username, email, password: rawPassword } = req.body as { name?: string; username?: string; email?: string; password?: unknown };
        if (!name || !username || !email || rawPassword === undefined || rawPassword === null) throw new BadRequestError("Missing required fields");
        const isProd = process.env.NODE_ENV === "production";

        if (isProd) {
            const decision = await aj.protect(req, { email });
            console.log("Arcjet decision:", decision);

            if (decision.isDenied()) {
                if (decision.reason.isEmail()) {
                    return res.status(400).json({ error: "Invalid email" });
                } else {
                    return res.status(403).json({ error: "Forbidden" });
                }
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

        // Issue tokens
        const { token: accessToken, jti: accessJti } = signAccessToken({ id: user.id, role: user.role });
        const { token: refreshToken, jti: refreshJti } = await signAndStoreRefreshToken(user.id);

        // Create session record in database
        const sessionDeviceInfo = await getSessionDeviceInfo(req);
        const expiresAt = new Date();
        expiresAt.setSeconds(expiresAt.getSeconds() + parseInt(process.env.ACCESS_TOKEN_TTL_SEC || "900", 10));
        const refreshExpiresAt = new Date();
        refreshExpiresAt.setSeconds(refreshExpiresAt.getSeconds() + parseInt(process.env.REFRESH_TOKEN_TTL_SEC || "2592000", 10));

        // Extract device info and create device record
        const deviceInfo = extractDeviceInfo(req);
        let deviceId: string | null = null;

        // Create device record for new user - use device name from headers if available
        const deviceName = sessionDeviceInfo.deviceName || extractDeviceName(sessionDeviceInfo.userAgent);
        const newDevice = await prisma.userDevice.create({
            data: {
                userId: user.id,
                deviceFingerprint: deviceInfo.fingerprint,
                deviceName: deviceName,
                platform: sessionDeviceInfo.platform as any,
                ipAddress: sessionDeviceInfo.ipAddress,
                isTrusted: true, // New registrations are trusted by default
                lastLoginAt: new Date(),
            },
        });
        deviceId = newDevice.id;

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

        // email verification OTP
        const otp = await createAndStoreOtp(`email:${email}`);
        // Get user language preference (default to English for new users)
        const userLanguage = 'en'; // New users don't have preferences yet
        // Send OTP via email (non-blocking)
        sendVerificationOTP(email, otp, name, userLanguage).catch(console.error);

        res.status(201).json({
            user: { id: user.id, name: user.name, email: user.email, verified: user.verified },
            accessToken,
            refreshToken,
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
                isActive: true,
                deletedAt: true,
            },
        });

        if (!user || !user.password) throw new UnauthorizedError("Invalid credentials");

        // Check if account is deleted (before password check for security)
        if (user.deletedAt) {
            throw new UnauthorizedError("Account has been deleted");
        }

        // Verify password first
        const ok = await bcrypt.compare(password, user.password);
        if (!ok) throw new UnauthorizedError("Invalid credentials");

        // Track if account was reactivated during login
        let wasReactivated = false;

        // If account is deactivated but password is correct, reactivate it automatically
        if (!user.isActive) {
            await prisma.user.update({
                where: { id: user.id },
                data: { isActive: true },
            });
            wasReactivated = true;
            // Update user object for rest of login flow
            user.isActive = true;
        }

        //  Unverified users: block login
        if (!user.verified) {
            return res.status(403).json({
                error: "Account not verified",
                message: wasReactivated
                    ? "Account reactivated. Please verify your account before logging in. Check your email for the verification OTP."
                    : "Please verify your account before logging in. Check your email for the verification OTP.",
                verified: false,
                requiresVerification: true,
                accountReactivated: wasReactivated,
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
                        ipAddress: deviceInfo.ipAddress,
                        platform: deviceInfo.platform,
                        // Device not verified yet
                        lastLoginAt: null,
                        isTrusted: false,
                    },
                });

                // Create device verification OTP
                const otp = await createAndStoreOtp(`device:${user.id}:${deviceInfo.fingerprint}`);
                // Get user language preference
                const userLanguage = await getUserLanguage(user.id);
                // Send OTP via email (non-blocking)
                sendDeviceVerificationOTP(user.email, otp, user.name, userLanguage).catch(console.error);

                return res.status(403).json({
                    error: "New device detected",
                    message: wasReactivated
                        ? "Account reactivated. A new device has been detected. Your account has been blocked until device verification is completed. Please check your email for verification code."
                        : "A new device has been detected. Your account has been blocked until device verification is completed. Please check your email for verification code.",
                    deviceBlocked: true,
                    requiresDeviceVerification: true,
                    deviceFingerprint: deviceInfo.fingerprint,
                    accountReactivated: wasReactivated,
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
                    // Device verified for first time
                    lastLoginAt: new Date(),
                    isTrusted: true, // First 2 devices are trusted by default
                },
            });
        } else {
            // Device exists, update login count and last login
            existingDevice = await prisma.userDevice.update({
                where: { id: existingDevice.id },
                data: {
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
                    message: wasReactivated
                        ? "Account reactivated. This device needs to be verified. Please check your email for verification code."
                        : "This device needs to be verified. Please check your email for verification code.",
                    deviceBlocked: true,
                    requiresDeviceVerification: true,
                    deviceFingerprint: deviceInfo.fingerprint,
                    accountReactivated: wasReactivated,
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
                // Keep local state in sync with DB so downstream checks behave correctly
                user.deviceBlocked = false;
                user.pendingDeviceFingerprint = null;
            }
        }

        // If account is blocked and this is not the pending device, block login
        if (user.deviceBlocked && user.pendingDeviceFingerprint !== deviceInfo.fingerprint) {
            return res.status(403).json({
                error: "Account blocked",
                message: wasReactivated
                    ? "Account reactivated. Your account has been blocked due to a new device login. Please verify the pending device first."
                    : "Your account has been blocked due to a new device login. Please verify the pending device first.",
                deviceBlocked: true,
                requiresDeviceVerification: true,
                accountReactivated: wasReactivated,
            });
        }

        // Check if 2FA is enabled (do this after device check)
        if (user.twoFactorEnabled) {
            // 2FA is enabled - issue temporary access token for 2FA verification
            const { token: tempAccessToken, jti: tempAccessJti } = signAccessToken({ id: user.id, role: user.role });

            // Create temporary session for 2FA verification (will be replaced after 2FA succeeds)
            const sessionDeviceInfo = await getSessionDeviceInfo(req);
            const expiresAt = new Date();
            expiresAt.setSeconds(expiresAt.getSeconds() + parseInt(process.env.ACCESS_TOKEN_TTL_SEC || "900", 10));

            // Find or get device ID
            let deviceId: string | null = null;
            if (existingDevice) {
                deviceId = existingDevice.id;
            } else {
                // Device was just created, find it
                const device = await prisma.userDevice.findUnique({
                    where: {
                        userId_deviceFingerprint: {
                            userId: user.id,
                            deviceFingerprint: deviceInfo.fingerprint,
                        },
                    },
                });
                if (device) {
                    deviceId = device.id;
                }
            }

            // Create temporary session (no refresh token yet, will be added after 2FA)
            // Note: We need to create a session with null refreshToken, but createSession requires a string
            // So we'll create it directly with Prisma
            await prisma.session.create({
                data: {
                    userId: user.id,
                    deviceId: deviceId,
                    sessionToken: tempAccessJti,
                    refreshToken: null, // Will be updated after 2FA verification
                    ipAddress: sessionDeviceInfo.ipAddress,
                    userAgent: sessionDeviceInfo.userAgent,
                    location: sessionDeviceInfo.location,
                    expiresAt: expiresAt,
                    refreshExpiresAt: null, // Will be set after 2FA verification
                    isActive: true,
                    isRevoked: false,
                    lastActivityAt: new Date(),
                },
            });

            return res.status(200).json({
                accessToken: tempAccessToken,
                message: wasReactivated
                    ? "Account reactivated. 2FA verification required"
                    : "2FA verification required",
                requires2FA: true,
                accountReactivated: wasReactivated,
            });
        }

        // Verified but not onboarded: issue token but flag the state
        const { token: accessToken, jti: accessJti } = signAccessToken({ id: user.id, role: user.role });
        const { token: refreshToken, jti: refreshJti } = await signAndStoreRefreshToken(user.id);

        // Create session record in database
        const sessionDeviceInfo = await getSessionDeviceInfo(req);
        const expiresAt = new Date();
        expiresAt.setSeconds(expiresAt.getSeconds() + parseInt(process.env.ACCESS_TOKEN_TTL_SEC || "900", 10));
        const refreshExpiresAt = new Date();
        refreshExpiresAt.setSeconds(refreshExpiresAt.getSeconds() + parseInt(process.env.REFRESH_TOKEN_TTL_SEC || "2592000", 10));

        // Find or get device ID
        let deviceId: string | null = null;
        if (existingDevice) {
            deviceId = existingDevice.id;
        } else {
            // Device was just created, find it
            const device = await prisma.userDevice.findUnique({
                where: {
                    userId_deviceFingerprint: {
                        userId: user.id,
                        deviceFingerprint: deviceInfo.fingerprint,
                    },
                },
            });
            if (device) {
                deviceId = device.id;
            }
        }

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
                twoFactorEnabled: user.twoFactorEnabled,
            },
            accessToken,
            refreshToken,
            requiresOnboarding: !user.onboardingCompleted,
            accountReactivated: wasReactivated,
            message: wasReactivated
                ? "Account reactivated successfully. Welcome back!"
                : undefined,
        });
    } catch (err) {
        next(err);
    }
};

export const logoutUser = async (req: Request, res: Response, next: NextFunction) => {
    try {
        if (!req.user) {
            throw new UnauthorizedError("User not authenticated");
        }

        const userId = req.user.id;
        const sessionTokenJti = req.user.jti;

        // Find the session by sessionToken JTI
        const session = await prisma.session.findFirst({
            where: {
                userId: userId,
                sessionToken: sessionTokenJti,
            },
            select: {
                id: true,
                sessionToken: true,
                refreshToken: true,
            },
        });

        if (!session) {
            // Session not found, but still return success (already logged out)
            return res.json({ 
                message: "Logged out successfully",
                sessionsDeleted: 0,
                debug: {
                    userId,
                    sessionTokenJti,
                    sessionFound: false
                }
            });
        }

        // Use the working revokeSession utility function
        await revokeSession(session.id, userId, sessionTokenJti);

        res.json({ 
            message: "Logged out successfully",
            sessionsDeleted: 1,
            debug: {
                userId,
                sessionId: session.id,
                sessionFound: true
            }
        });
    } catch (err) {
        // Return error details in response for debugging
        if (err instanceof Error) {
            return res.status(500).json({
                error: "Logout failed",
                message: err.message,
                stack: process.env.NODE_ENV === "development" ? err.stack : undefined
            });
        }
        next(err);
    }
}

export const refreshToken = async (req: Request, res: Response, next: NextFunction) => {
    try {
        // Extract refresh token from request (header or body)
        let refreshToken = getRefreshTokenFromRequest(req);

        // Fallback: Check request body (common for mobile apps)
        if (!refreshToken && req.body?.refreshToken) {
            refreshToken = req.body.refreshToken;
        }

        if (!refreshToken) {
            throw new UnauthorizedError("Refresh token is required. Send it in x-refresh-token header or request body.");
        }

        // Verify the refresh token
        let payload;
        try {
            payload = await verifyRefreshToken(refreshToken);
        } catch (error) {
            console.error("[Refresh Token] Verification failed:", error instanceof Error ? error.message : error);
            if (error instanceof Error) {
                if (error.name === "JsonWebTokenError") {
                    throw new UnauthorizedError("Invalid refresh token format");
                }
                if (error.name === "TokenExpiredError") {
                    throw new UnauthorizedError("Refresh token has expired");
                }
                if (error.message.includes("revoked") || error.message.includes("expired")) {
                    throw new UnauthorizedError("Refresh token has been revoked or expired");
                }
            }
            throw error;
        }
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
                bio: true,
                goals: true,
                newsletterEnabled: true,
                profileImg: true,
                isActive: true,
                deletedAt: true,
                twoFactorEnabled: true,
            },
        });

        if (!user) {
            // User doesn't exist anymore, revoke the token
            await revokeRefreshTokenByJti(payload.jti);
            throw new UnauthorizedError("User not found");
        }

        // Check if account is deleted
        if (user.deletedAt) {
            await revokeRefreshTokenByJti(payload.jti);
            throw new UnauthorizedError("Account has been deleted");
        }

        // Check if account is deactivated
        if (!user.isActive) {
            await revokeRefreshTokenByJti(payload.jti);
            throw new UnauthorizedError("Account is deactivated");
        }

        // Check if user account is verified
        if (!user.verified) {
            throw new UnauthorizedError("Account not verified");
        }

        // Find session by old refresh token JTI BEFORE rotating token
        const session = await prisma.session.findFirst({
            where: {
                userId: user.id,
                refreshToken: payload.jti, // Old refresh token JTI
            },
        });

        // Generate new access token with current user role
        const { token: accessToken, jti: newAccessJti } = signAccessToken({ id: user.id, role: user.role });

        // Rotate refresh token for security (revoke old, create new)
        const { token: newRefreshToken, jti: newRefreshJti } = await rotateRefreshToken(payload.jti, user.id);

        // Update session record with new tokens if session exists
        if (session) {
            const expiresAt = new Date();
            expiresAt.setSeconds(expiresAt.getSeconds() + parseInt(process.env.ACCESS_TOKEN_TTL_SEC || "900", 10));
            const refreshExpiresAt = new Date();
            refreshExpiresAt.setSeconds(refreshExpiresAt.getSeconds() + parseInt(process.env.REFRESH_TOKEN_TTL_SEC || "2592000", 10));

            await prisma.session.update({
                where: { id: session.id },
                data: {
                    sessionToken: newAccessJti, // Update with new access token JTI
                    refreshToken: newRefreshJti, // Update with new refresh token JTI
                    expiresAt: expiresAt,
                    refreshExpiresAt: refreshExpiresAt,
                    lastActivityAt: new Date(),
                },
            });
        }

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
                twoFactorEnabled: user.twoFactorEnabled,
            },
            accessToken,
            refreshToken: newRefreshToken,
        });
    } catch (err) {
        // Log error for debugging
        console.error("[Refresh Token] Error:", err instanceof Error ? err.message : err);
        if (err instanceof Error && err.stack) {
            console.error("[Refresh Token] Stack:", err.stack);
        }
        next(err);
    }
}

export const getMyProfile = async (req: Request, res: Response, next: NextFunction) => {
    try {
        // Extract user ID from req.user (populated by authenticate middleware)
        const userId = req.user?.id;

        if (!userId) {
            throw new UnauthorizedError("User not authenticated");
        }

        // Query database using Prisma to get user profile
        const user = await prisma.user.findUnique({
            where: { id: userId },
            select: {
                id: true,
                name: true,
                username: true,
                email: true,
                verified: true,
                onboardingCompleted: true,
                role: true,
                profileImg: true,
                isActive: true,
                lastLoginAt: true,
                createdAt: true,
                updatedAt: true,
                deviceBlocked: true,
                bio: true,
                goals: true,
                newsletterEnabled: true,
                // Include user preferences
                preferences: {
                    select: {
                        language: true,
                        themePreference: true,
                        notifications: true
                    },
                },
                // Explicitly exclude sensitive fields
                password: false,
                deletedAt: false,
                pendingDeviceFingerprint: false,
                twoFactorSecret: false,
                twoFactorBackupCodes: false,
            },
        });

        // Return 404 error if user not found
        if (!user) {
            return res.status(404).json({ error: "User not found" });
        }

        // Return user profile data in response
        res.json({ user });
    } catch (err) {
        next(err);
    }
};
