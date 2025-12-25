import { Request, Response, NextFunction } from "express";
import bcrypt from "bcrypt";
import prisma from "../libs/prisma";
import { BadRequestError, UnauthorizedError } from "../utils/errors";
import { revokeAllUserRefreshTokens, signAccessToken, signAndStoreRefreshToken } from "../utils/tokens";
import { deleteImageFromCloudinary } from "../utils/cloudinaryUpload";
import { extractDeviceInfo, extractDeviceName } from "../utils/device";
import { createSession, getSessionDeviceInfo } from "../utils/sessions";

/**
 * Deactivate user account
 * Sets isActive to false and revokes all refresh tokens
 */
export const deactivateAccount = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const userId = req.user?.id;

        if (!userId) {
            throw new UnauthorizedError("Authentication required");
        }

        // Get user to verify they exist and are not already deactivated
        const user = await prisma.user.findUnique({
            where: { id: userId },
            select: {
                id: true,
                isActive: true,
                deletedAt: true,
            },
        });

        if (!user) {
            throw new UnauthorizedError("User not found");
        }

        if (!user.isActive) {
            return res.status(400).json({
                error: "Account already deactivated",
                message: "Your account is already deactivated.",
            });
        }

        if (user.deletedAt) {
            throw new UnauthorizedError("Account has been deleted");
        }

        // Revoke all refresh tokens
        await revokeAllUserRefreshTokens(userId);

        // Update user account status
        await prisma.user.update({
            where: { id: userId },
            data: { isActive: false },
        });

        res.json({
            message: "Account deactivated successfully",
            deactivated: true,
        });
    } catch (err) {
        next(err);
    }
};

/**
 * Delete user account (soft delete)
 * Requires password confirmation for security
 */
export const deleteAccount = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const userId = req.user?.id;
        const { password } = req.body as { password?: string };

        if (!userId) {
            throw new UnauthorizedError("Authentication required");
        }

        if (!password) {
            throw new BadRequestError("Password confirmation is required");
        }

        // Get user with password to verify
        const user = await prisma.user.findUnique({
            where: { id: userId },
            select: {
                id: true,
                password: true,
                deletedAt: true,
            },
        });

        if (!user) {
            throw new UnauthorizedError("User not found");
        }

        if (user.deletedAt) {
            return res.status(400).json({
                error: "Account already deleted",
                message: "Your account has already been deleted.",
            });
        }

        // Verify password
        if (!user.password) {
            throw new BadRequestError("Password verification not available for this account");
        }

        const passwordValid = await bcrypt.compare(password, user.password);
        if (!passwordValid) {
            throw new UnauthorizedError("Invalid password");
        }

        // Revoke all refresh tokens
        await revokeAllUserRefreshTokens(userId);

        // Soft delete user account
        await prisma.user.update({
            where: { id: userId },
            data: {
                deletedAt: new Date(),
                isActive: false, // Also deactivate
            },
        });

        res.json({
            message: "Account deleted successfully",
            deleted: true,
        });
    } catch (err) {
        next(err);
    }
};


/**
 * Delete user profile image
 * Removes image from Cloudinary if applicable and sets profileImg to null
 */
export const deleteProfileImage = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const userId = req.user?.id;

        if (!userId) {
            throw new UnauthorizedError("Authentication required");
        }

        // Get user to check current profile image
        const user = await prisma.user.findUnique({
            where: { id: userId },
            select: { profileImg: true },
        });

        if (!user) {
            throw new UnauthorizedError("User not found");
        }

        if (!user.profileImg) {
            return res.status(200).json({
                message: "No profile image to delete",
            });
        }

        // If it's a Cloudinary URL, delete it from storage
        if (user.profileImg.includes("cloudinary.com")) {
            await deleteImageFromCloudinary(user.profileImg);
        }

        // Update user to remove profile image reference
        await prisma.user.update({
            where: { id: userId },
            data: { profileImg: null },
        });

        res.json({
            message: "Profile image deleted successfully",
        });
    } catch (err) {
        next(err);
    }
};

/**
 * Confirm account reactivation
 * User must call this endpoint with temp token to fully reactivate their account
 */
export const confirmReactivation = async (req: Request, res: Response, next: NextFunction) => {
    try {
        if (!req.user) {
            throw new UnauthorizedError("Authentication required");
        }

        const userId = req.user.id;

        // Fetch user to check current state
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
                twoFactorEnabled: true,
                isActive: true,
                deletedAt: true,
            },
        });

        if (!user) {
            throw new UnauthorizedError("User not found");
        }

        if (user.deletedAt) {
            throw new UnauthorizedError("Account has been deleted");
        }

        if (user.isActive) {
            // Account is already active
            return res.status(400).json({
                error: "Account already active",
                message: "Your account is already active",
            });
        }

        // Reactivate the account
        await prisma.user.update({
            where: { id: userId },
            data: { 
                isActive: true,
                lastLoginAt: new Date(),
            },
        });

        // Issue full access tokens
        const { token: accessToken, jti: accessJti } = signAccessToken({ id: user.id, role: user.role });
        const { token: refreshToken, jti: refreshJti } = await signAndStoreRefreshToken(user.id);

        // Create session record
        const sessionDeviceInfo = await getSessionDeviceInfo(req);
        const expiresAt = new Date();
        expiresAt.setSeconds(expiresAt.getSeconds() + parseInt(process.env.ACCESS_TOKEN_TTL_SEC || "900", 10));
        const refreshExpiresAt = new Date();
        refreshExpiresAt.setSeconds(refreshExpiresAt.getSeconds() + parseInt(process.env.REFRESH_TOKEN_TTL_SEC || "2592000", 10));

        // Extract device info
        const deviceInfo = extractDeviceInfo(req);
        let deviceId: string | null = null;

        // Find or create device
        const existingDevice = await prisma.userDevice.findUnique({
            where: {
                userId_deviceFingerprint: {
                    userId: user.id,
                    deviceFingerprint: deviceInfo.fingerprint,
                },
            },
        });

        if (existingDevice) {
            deviceId = existingDevice.id;
            await prisma.userDevice.update({
                where: { id: existingDevice.id },
                data: { lastLoginAt: new Date() },
            });
        } else {
            const deviceName = sessionDeviceInfo.deviceName || extractDeviceName(sessionDeviceInfo.userAgent);
            const newDevice = await prisma.userDevice.create({
                data: {
                    userId: user.id,
                    deviceFingerprint: deviceInfo.fingerprint,
                    deviceName: deviceName,
                    platform: sessionDeviceInfo.platform as any,
                    ipAddress: sessionDeviceInfo.ipAddress,
                    userAgent: sessionDeviceInfo.userAgent,
                    isTrusted: true,
                    lastLoginAt: new Date(),
                },
            });
            deviceId = newDevice.id;
        }

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

        res.json({
            message: "Account reactivated successfully. Welcome back!",
            accountReactivated: true,
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
