import { Request } from "express";
import prisma from "../libs/prisma";
import { signAccessToken, signAndStoreRefreshToken } from "../utils/tokens";
import { extractDeviceInfo, extractDeviceName } from "../utils/device";
import { createSession, getSessionDeviceInfo } from "../utils/sessions";

export interface SessionDeviceInfo {
    ipAddress: string;
    userAgent: string;
    location: string;
}

export interface TokenPair {
    accessToken: string;
    refreshToken: string;
    accessJti: string;
    refreshJti: string;
}

export interface SessionExpiry {
    expiresAt: Date;
    refreshExpiresAt: Date;
}

/**
 * Calculate session expiry dates based on environment variables
 */
export function calculateSessionExpiry(): SessionExpiry {
    const expiresAt = new Date();
    expiresAt.setSeconds(expiresAt.getSeconds() + parseInt(process.env.ACCESS_TOKEN_TTL_SEC || "900", 10));
    const refreshExpiresAt = new Date();
    refreshExpiresAt.setSeconds(refreshExpiresAt.getSeconds() + parseInt(process.env.REFRESH_TOKEN_TTL_SEC || "2592000", 10));
    
    return { expiresAt, refreshExpiresAt };
}

/**
 * Generate access and refresh tokens for a user
 */
export async function generateTokens(userId: string, userRole: string): Promise<TokenPair> {
    const { token: accessToken, jti: accessJti } = signAccessToken({ id: userId, role: userRole });
    const { token: refreshToken, jti: refreshJti } = await signAndStoreRefreshToken(userId);
    
    return { accessToken, refreshToken, accessJti, refreshJti };
}

/**
 * Find existing device or create a new one for the user
 */
export async function findOrCreateDevice(
    userId: string,
    deviceFingerprint: string,
    deviceName?: string,
    isTrusted: boolean = true,
    lastLoginAt: Date | null = new Date()
) {
    // Try to find existing device
    const existingDevice = await prisma.userDevice.findUnique({
        where: {
            userId_deviceFingerprint: {
                userId,
                deviceFingerprint,
            },
        },
    });

    if (existingDevice) {
        // Update existing device
        return await prisma.userDevice.update({
            where: { id: existingDevice.id },
            data: {
                lastLoginAt,
                isTrusted: existingDevice.isTrusted || isTrusted, // Don't downgrade trust
            },
        });
    } else {
        // Create new device
        return await prisma.userDevice.create({
            data: {
                userId,
                deviceFingerprint,
                deviceName: deviceName || "Unknown Device",
                isTrusted,
                lastLoginAt,
            },
        });
    }
}

/**
 * Create a complete user session with device and tokens
 */
export async function createDeviceAndSession(
    req: Request,
    userId: string,
    userRole: string,
    options: {
        deviceName?: string;
        isTrusted?: boolean;
        lastLoginAt?: Date | null;
    } = {}
) {
    const { deviceName, isTrusted = true, lastLoginAt = new Date() } = options;
    
    // Extract device info
    const deviceInfo = extractDeviceInfo(req, deviceName);
    const sessionDeviceInfo = await getSessionDeviceInfo(req);
    
    // Find or create device
    const device = await findOrCreateDevice(
        userId,
        deviceInfo.fingerprint,
        extractDeviceName(deviceInfo.userAgent) || deviceName,
        isTrusted,
        lastLoginAt
    );
    
    // Generate tokens
    const tokens = await generateTokens(userId, userRole);
    
    // Calculate expiry
    const { expiresAt, refreshExpiresAt } = calculateSessionExpiry();
    
    // Create session
    await createSession({
        userId,
        deviceId: device.id,
        sessionToken: tokens.accessJti,
        refreshTokenJti: tokens.refreshJti,
        ipAddress: sessionDeviceInfo.ipAddress,
        userAgent: sessionDeviceInfo.userAgent,
        location: sessionDeviceInfo.location,
        expiresAt,
        refreshExpiresAt,
    });
    
    return {
        device,
        tokens,
        sessionInfo: {
            expiresAt,
            refreshExpiresAt,
            sessionDeviceInfo,
        },
    };
}

/**
 * Create a temporary session for 2FA verification (no refresh token)
 */
export async function createTemporary2FASession(
    req: Request,
    userId: string,
    userRole: string,
    deviceId: string
) {
    // Generate temporary access token
    const { token: tempAccessToken, jti: tempAccessJti } = signAccessToken({ id: userId, role: userRole });
    
    // Get session device info
    const sessionDeviceInfo = await getSessionDeviceInfo(req);
    const { expiresAt } = calculateSessionExpiry();
    
    // Create temporary session (no refresh token yet)
    await prisma.session.create({
        data: {
            userId,
            deviceId,
            sessionToken: tempAccessJti,
            refreshToken: null, // Will be updated after 2FA verification
            ipAddress: sessionDeviceInfo.ipAddress,
            userAgent: sessionDeviceInfo.userAgent,
            location: sessionDeviceInfo.location,
            expiresAt,
            refreshExpiresAt: null, // Will be set after 2FA verification
            isActive: true,
            isRevoked: false,
            lastActivityAt: new Date(),
        },
    });
    
    return {
        tempAccessToken,
        tempAccessJti,
        expiresAt,
        sessionDeviceInfo,
    };
}
