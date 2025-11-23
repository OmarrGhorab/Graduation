import { Request, Response, NextFunction } from "express";
import prisma from "../libs/prisma";
import { UnauthorizedError, BadRequestError } from "../utils/errors";
import { revokeSession, revokeAllUserSessions } from "../utils/sessions";
import { getAccessTokenFromRequest } from "../utils/cookies";
import { verifyAccessToken } from "../utils/tokens";
import { clearAuthCookies } from "../utils/cookies";

/**
 * Helper function to extract current session token from request
 * Returns null if token is missing or invalid
 */
async function getCurrentSessionToken(req: Request): Promise<string | null> {
    const currentToken = getAccessTokenFromRequest(req);
    if (!currentToken) {
        return null;
    }

    try {
        const payload = await verifyAccessToken(currentToken);
        return payload.jti;
    } catch {
        return null;
    }
}

/**
 * Get all sessions for the authenticated user
 * Returns active sessions first, then inactive, ordered by lastActivityAt
 */
export const getSessions = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const userId = req.user?.id;

        if (!userId) {
            throw new UnauthorizedError("Authentication required");
        }

        // Get current session token to mark current session
        const currentSessionToken = await getCurrentSessionToken(req);

        // Get all sessions with device info
        const sessions = await prisma.session.findMany({
            where: {
                userId: userId,
            },
            include: {
                device: {
                    select: {
                        deviceName: true,
                        platform: true,
                        ipAddress: true,
                        userAgent: true,
                        isTrusted: true,
                        lastLoginAt: true,
                    },
                },
            },
            orderBy: [
                { isActive: "desc" }, // Active first
                { lastActivityAt: "desc" }, // Most recent first
            ],
        });

        // Format sessions response
        const formattedSessions = sessions.map((session) => {
            const isCurrent = session.sessionToken === currentSessionToken;
            const isExpired = new Date(session.expiresAt) < new Date();
            const isActive = session.isActive && !session.isRevoked && !isExpired;

            return {
                id: session.id,
                deviceName: session.device?.deviceName || "Unknown Device",
                platform: session.device?.platform || null,
                ipAddress: session.ipAddress || session.device?.ipAddress || null,
                location: session.location || null,
                isActive: isActive,
                isCurrent: isCurrent,
                isRevoked: session.isRevoked,
                isExpired: isExpired,
                lastActivityAt: session.lastActivityAt,
                createdAt: session.createdAt,
                expiresAt: session.expiresAt,
                revokedAt: session.revokedAt,
            };
        });

        res.json({
            sessions: formattedSessions,
            totalSessions: formattedSessions.length,
            activeSessions: formattedSessions.filter((s) => s.isActive).length,
        });
    } catch (err) {
        next(err);
    }
};

/**
 * Revoke a specific session
 * If it's the current session, logs the user out
 */
export const revokeSessionById = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const userId = req.user?.id;
        const { sessionId } = req.params;

        if (!userId) {
            throw new UnauthorizedError("Authentication required");
        }

        if (!sessionId) {
            throw new BadRequestError("Session ID is required");
        }

        // Get current session token to check if we're revoking the current session
        const currentSessionToken = await getCurrentSessionToken(req);

        const { isCurrentSession } = await revokeSession(sessionId, userId, currentSessionToken);

        // If revoking current session, log user out
        if (isCurrentSession) {
            clearAuthCookies(res);
            return res.json({
                message: "Session revoked successfully. You have been logged out.",
                revoked: true,
                loggedOut: true,
            });
        }

        res.json({
            message: "Session revoked successfully",
            revoked: true,
            loggedOut: false,
        });
    } catch (err) {
        if (err instanceof Error) {
            if (err.message === "Session not found") {
                return next(new BadRequestError("Session not found"));
            }
            if (err.message === "Unauthorized to revoke this session") {
                return next(new UnauthorizedError("Unauthorized to revoke this session"));
            }
        }
        next(err);
    }
};

/**
 * Revoke all sessions
 * Query parameter `includeCurrent=true` to also revoke current session (logs user out)
 * Default behavior: excludes current session (user stays logged in)
 */
export const revokeAllSessions = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const userId = req.user?.id;

        if (!userId) {
            throw new UnauthorizedError("Authentication required");
        }

        // Get query parameter to determine if we should include current session
        const includeCurrent = req.query.includeCurrent === "true" || req.body?.includeCurrent === true;

        // Get current session token
        const currentSessionToken = await getCurrentSessionToken(req);
        if (!currentSessionToken) {
            throw new UnauthorizedError("Current session token not found");
        }

        // Revoke sessions using utility function
        const { deletedCount, wasCurrentIncluded } = await revokeAllUserSessions(
            userId,
            currentSessionToken,
            includeCurrent
        );

        // If current session was included, log user out
        if (wasCurrentIncluded) {
            clearAuthCookies(res);
            return res.json({
                message: `Revoked ${deletedCount} session(s) successfully. You have been logged out.`,
                revokedCount: deletedCount,
                loggedOut: true,
            });
        }

        res.json({
            message: deletedCount > 0 
                ? `Revoked ${deletedCount} session(s) successfully`
                : "No sessions to revoke.",
            revokedCount: deletedCount,
            loggedOut: false,
        });
    } catch (err) {
        next(err);
    }
};

