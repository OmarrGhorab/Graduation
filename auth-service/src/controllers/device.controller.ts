import { Request, Response, NextFunction } from "express";
import prisma from "../libs/prisma";
import { BadRequestError, UnauthorizedError } from "../utils/errors";
import { signAccessToken, signAndStoreRefreshToken } from "../utils/tokens";
import { createAndStoreOtp, verifyOtp } from "../utils/otp";
import { extractDeviceInfo } from "../utils/device";
import { createSession, getSessionDeviceInfo } from "../utils/sessions";
import { sendDeviceVerificationOTP } from "../utils/email";
import { getUserLanguage } from "../utils/userLanguage";
import { publishNotification } from "../utils/notifications-client";
import { debugLog } from "../utils/debug-logger";

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

        // Enforce account state before allowing device verification
        if (user.deletedAt) {
            throw new UnauthorizedError("Account has been deleted");
        }

        if (!user.isActive) {
            throw new UnauthorizedError("Account is deactivated");
        }

        if (!user.verified) {
            throw new UnauthorizedError("Account not verified");
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

        // SECURITY: Notify user that a new device was successfully added
        const notificationData = {
            type: "security_device_verified",
            title: "New Device Added",
            body: `A new device (${device.deviceName || "Unknown"}) has been successfully verified and added to your account.`,
            deviceName: device.deviceName,
            platform: device.platform,
            timestamp: new Date().toISOString(),
            securityTip: "If you didn't add this device, please change your password and review your account security immediately.",
        };

        // Send notification (BLOCKING - wait for it to complete)
        console.log(`[Security] 🔓 Sending device verified notification to user ${user.id}`);
        try {
            await publishNotification(user.id, notificationData);
            console.log(`[Security] ✓ Device verified notification sent successfully to user ${user.id}`);
        } catch (err) {
            console.error("[Security Notification] ✗ Failed to send device verified alert:", err);
        }

        // Check if 2FA is enabled
        if (user.twoFactorEnabled) {
            // 2FA is enabled - issue temporary access token for 2FA verification
            const { token: tempAccessToken, jti: tempAccessJti } = signAccessToken({ id: user.id, role: user.role });

            // Create temporary session for 2FA verification (will be replaced after 2FA succeeds)
            const sessionDeviceInfo = await getSessionDeviceInfo(req);
            const expiresAt = new Date();
            expiresAt.setSeconds(expiresAt.getSeconds() + parseInt(process.env.ACCESS_TOKEN_TTL_SEC || "900", 10));

            // Create temporary session (no refresh token yet, will be added after 2FA)
            await prisma.session.create({
                data: {
                    userId: user.id,
                    deviceId: device.id,
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
                message: "Device verified successfully. 2FA verification required.",
                deviceVerified: true,
                requires2FA: true,
                accessToken: tempAccessToken,
                emailOrUsername: emailOrUsername,
            });
        }

        // Issue tokens
        const { token: accessToken, jti: accessJti } = signAccessToken({ id: user.id, role: user.role });
        const { token: refreshToken, jti: refreshJti } = await signAndStoreRefreshToken(user.id);

        // Create session record in database
        const sessionDeviceInfo = await getSessionDeviceInfo(req);
        const expiresAt = new Date();
        expiresAt.setSeconds(expiresAt.getSeconds() + parseInt(process.env.ACCESS_TOKEN_TTL_SEC || "900", 10));
        const refreshExpiresAt = new Date();
        refreshExpiresAt.setSeconds(refreshExpiresAt.getSeconds() + parseInt(process.env.REFRESH_TOKEN_TTL_SEC || "2592000", 10));

        await createSession({
            userId: user.id,
            deviceId: device.id,
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
                twoFactorEnabled: user.twoFactorEnabled,
            },
            accessToken,
            refreshToken,
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
        // Get user language preference
        const userLanguage = await getUserLanguage(user.id);
        // Send OTP via email (non-blocking)
        debugLog(`[Device] 📧 Resending verification OTP to ${user.email}`);
        sendDeviceVerificationOTP(user.email, otp, user.name, userLanguage)
            .then(sent => {
                if (sent) debugLog(`[Device] ✓ Verification OTP resent successfully`);
            })
            .catch(err => debugLog(`[Device] ✗ Error resending OTP`, { error: String(err) }));

        return res.json({
            message: "Device verification OTP has been sent.",
            // Expose OTP in non-production for testing
            otp: process.env.NODE_ENV === "production" ? undefined : otp,
        });
    } catch (err) {
        next(err);
    }
};
