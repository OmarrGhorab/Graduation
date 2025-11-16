import { Request, Response, NextFunction } from "express";
import prisma from "../libs/prisma";
import { UnauthorizedError } from "../utils/errors";
import { extractDeviceInfo } from "../utils/device";

/**
 * Get user activity information
 * Shows last activity time, current device info, and active session count
 */
export const getActivity = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const userId = req.user?.id;

        if (!userId) {
            throw new UnauthorizedError("Authentication required");
        }

        // Get most recent session activity from active sessions only
        const mostRecentSession = await prisma.session.findFirst({
            where: {
                userId: userId,
                isActive: true,
                isRevoked: false,
                expiresAt: {
                    gt: new Date(), // Not expired
                },
            },
            orderBy: {
                lastActivityAt: "desc",
            },
            select: {
                lastActivityAt: true,
            },
        });

        // Get current device info from request
        const deviceInfo = extractDeviceInfo(req);
        const currentDevice = {
            deviceName: deviceInfo.deviceName || "Unknown Device",
            platform: deviceInfo.platform,
            ipAddress: deviceInfo.ipAddress,
            location: null, // Can be enhanced with geo IP lookup
        };

        // Count active sessions
        const activeSessionsCount = await prisma.session.count({
            where: {
                userId: userId,
                isActive: true,
                isRevoked: false,
                expiresAt: {
                    gt: new Date(),
                },
            },
        });

        res.json({
            lastActivityAt: mostRecentSession?.lastActivityAt || null,
            currentDevice: currentDevice,
            totalActiveSessions: activeSessionsCount,
        });
    } catch (err) {
        next(err);
    }
};

