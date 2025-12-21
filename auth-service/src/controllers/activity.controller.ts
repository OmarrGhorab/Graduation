import { Request, Response, NextFunction } from "express";
import prisma from "../libs/prisma";
import { UnauthorizedError } from "../utils/errors";
import { getDeviceInfoFromRequest } from "../middleware/deviceInfo.middleware";

/**
 * Parse user agent string to extract device info
 */
function parseUserAgent(userAgent: string | null | undefined): {
    browser: string | null;
    browserVersion: string | null;
    os: string | null;
    osVersion: string | null;
    deviceType: string;
} {
    if (!userAgent) {
        return {
            browser: null,
            browserVersion: null,
            os: null,
            osVersion: null,
            deviceType: "unknown",
        };
    }

    let browser: string | null = null;
    let browserVersion: string | null = null;
    let os: string | null = null;
    let osVersion: string | null = null;
    let deviceType = "desktop";

    // Detect browser and version
    if (userAgent.includes("Chrome") && !userAgent.includes("Edg")) {
        browser = "Chrome";
        const match = userAgent.match(/Chrome\/(\d+\.?\d*)/);
        browserVersion = match ? match[1] : null;
    } else if (userAgent.includes("Safari") && !userAgent.includes("Chrome")) {
        browser = "Safari";
        const match = userAgent.match(/Version\/(\d+\.?\d*)/);
        browserVersion = match ? match[1] : null;
    } else if (userAgent.includes("Firefox")) {
        browser = "Firefox";
        const match = userAgent.match(/Firefox\/(\d+\.?\d*)/);
        browserVersion = match ? match[1] : null;
    } else if (userAgent.includes("Edg")) {
        browser = "Edge";
        const match = userAgent.match(/Edg\/(\d+\.?\d*)/);
        browserVersion = match ? match[1] : null;
    } else if (userAgent.includes("Opera") || userAgent.includes("OPR")) {
        browser = "Opera";
        const match = userAgent.match(/(?:Opera|OPR)\/(\d+\.?\d*)/);
        browserVersion = match ? match[1] : null;
    }

    // Detect OS and version
    if (userAgent.includes("Windows NT 10")) {
        os = "Windows";
        osVersion = "10/11";
    } else if (userAgent.includes("Windows NT 6.3")) {
        os = "Windows";
        osVersion = "8.1";
    } else if (userAgent.includes("Windows NT 6.1")) {
        os = "Windows";
        osVersion = "7";
    } else if (userAgent.includes("Windows")) {
        os = "Windows";
    } else if (userAgent.includes("Mac OS X")) {
        os = "macOS";
        const match = userAgent.match(/Mac OS X (\d+[._]\d+)/);
        osVersion = match ? match[1].replace("_", ".") : null;
    } else if (userAgent.includes("Android")) {
        os = "Android";
        deviceType = "mobile";
        const match = userAgent.match(/Android (\d+\.?\d*)/);
        osVersion = match ? match[1] : null;
    } else if (userAgent.includes("iPhone")) {
        os = "iOS";
        deviceType = "mobile";
        const match = userAgent.match(/iPhone OS (\d+[._]\d+)/);
        osVersion = match ? match[1].replace("_", ".") : null;
    } else if (userAgent.includes("iPad")) {
        os = "iPadOS";
        deviceType = "tablet";
        const match = userAgent.match(/CPU OS (\d+[._]\d+)/);
        osVersion = match ? match[1].replace("_", ".") : null;
    } else if (userAgent.includes("Linux")) {
        os = "Linux";
    }

    return { browser, browserVersion, os, osVersion, deviceType };
}

/**
 * Get user activity information
 * Shows detailed activity info, sessions summary, devices, and recent activity
 */
export const getActivity = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const userId = req.user?.id;

        if (!userId) {
            throw new UnauthorizedError("Authentication required");
        }

        // Get user's last login time
        const user = await prisma.user.findUnique({
            where: { id: userId },
            select: {
                lastLoginAt: true,
                createdAt: true,
            },
        });

        // Get all active sessions with device info
        const activeSessions = await prisma.session.findMany({
            where: {
                userId: userId,
                isActive: true,
                isRevoked: false,
                expiresAt: {
                    gt: new Date(),
                },
            },
            include: {
                device: {
                    select: {
                        deviceName: true,
                        platform: true,
                        isTrusted: true,
                    },
                },
            },
            orderBy: {
                lastActivityAt: "desc",
            },
        });

        // Get current device info from request headers
        const deviceInfo = getDeviceInfoFromRequest(req);
        const parsedUA = parseUserAgent(deviceInfo.userAgent);

        const currentDevice = {
            // Use client-provided name/model if available
            deviceName: deviceInfo.deviceName || deviceInfo.deviceModel || `${parsedUA.browser || "Unknown"} on ${parsedUA.os || "Unknown"}`,
            deviceModel: deviceInfo.deviceModel,
            platform: deviceInfo.platform,
            browser: parsedUA.browser,
            browserVersion: parsedUA.browserVersion,
            os: deviceInfo.osVersion || (parsedUA.os ? `${parsedUA.os}${parsedUA.osVersion ? ` ${parsedUA.osVersion}` : ""}` : null),
            deviceType: parsedUA.deviceType,
            ipAddress: deviceInfo.ipAddress,
            // Client-provided location and timezone
            location: deviceInfo.location,
            timezone: deviceInfo.timezone,
            appVersion: deviceInfo.appVersion,
        };

        // Get trusted devices count
        const trustedDevicesCount = await prisma.userDevice.count({
            where: {
                userId: userId,
                isTrusted: true,
            },
        });

        // Get all user devices
        const devices = await prisma.userDevice.findMany({
            where: { userId: userId },
            select: {
                id: true,
                deviceName: true,
                platform: true,
                isTrusted: true,
                lastLoginAt: true,
                createdAt: true,
            },
            orderBy: {
                lastLoginAt: "desc",
            },
            take: 10,
        });

        // Build sessions summary by platform
        const sessionsByPlatform: Record<string, number> = {};
        activeSessions.forEach((session) => {
            const platform = session.device?.platform || "UNKNOWN";
            sessionsByPlatform[platform] = (sessionsByPlatform[platform] || 0) + 1;
        });

        // Get recent session activity (last 5 sessions including inactive)
        const recentSessions = await prisma.session.findMany({
            where: { userId: userId },
            include: {
                device: {
                    select: {
                        deviceName: true,
                        platform: true,
                    },
                },
            },
            orderBy: {
                lastActivityAt: "desc",
            },
            take: 5,
        });

        const recentActivity = recentSessions.map((session) => {
            const isExpired = new Date(session.expiresAt) < new Date();
            const isActive = session.isActive && !session.isRevoked && !isExpired;
            
            return {
                sessionId: session.id,
                deviceName: session.device?.deviceName || "Unknown Device",
                platform: session.device?.platform || null,
                ipAddress: session.ipAddress,
                location: session.location,
                lastActivityAt: session.lastActivityAt,
                createdAt: session.createdAt,
                status: isActive ? "active" : session.isRevoked ? "revoked" : isExpired ? "expired" : "inactive",
            };
        });

        res.json({
            // Account info
            account: {
                lastLoginAt: user?.lastLoginAt || null,
                accountCreatedAt: user?.createdAt || null,
            },
            // Current request info
            currentDevice,
            // Sessions summary
            sessions: {
                totalActive: activeSessions.length,
                byPlatform: sessionsByPlatform,
                mostRecentActivity: activeSessions[0]?.lastActivityAt || null,
            },
            // Devices summary
            devices: {
                total: devices.length,
                trusted: trustedDevicesCount,
                list: devices.map((d) => ({
                    id: d.id,
                    name: d.deviceName,
                    platform: d.platform,
                    isTrusted: d.isTrusted,
                    lastLoginAt: d.lastLoginAt,
                })),
            },
            // Recent activity
            recentActivity,
        });
    } catch (err) {
        next(err);
    }
};

