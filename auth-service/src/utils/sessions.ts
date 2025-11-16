import { Request } from "express";
import prisma from "../libs/prisma";
import { extractDeviceInfo } from "./device";
import { revokeRefreshTokenByJti } from "./tokens";

interface CreateSessionParams {
  userId: string;
  deviceId: string | null;
  sessionToken: string; // JWT jti (access token)
  refreshTokenJti: string; // Refresh token jti
  ipAddress: string | null;
  userAgent: string | null;
  location?: string | null;
  expiresAt: Date;
  refreshExpiresAt?: Date | null;
}

/**
 * Create a new session record in the database
 */
export async function createSession(params: CreateSessionParams) {
  const session = await prisma.session.create({
    data: {
      userId: params.userId,
      deviceId: params.deviceId,
      sessionToken: params.sessionToken,
      refreshToken: params.refreshTokenJti,
      ipAddress: params.ipAddress,
      userAgent: params.userAgent,
      location: params.location || null,
      expiresAt: params.expiresAt,
      refreshExpiresAt: params.refreshExpiresAt || null,
      isActive: true,
      isRevoked: false,
      lastActivityAt: new Date(),
    },
  });

  return session;
}

/**
 * Update session activity timestamp
 * Called when an authenticated request is made
 */
export async function updateSessionActivity(sessionToken: string) {
  await prisma.session.updateMany({
    where: {
      sessionToken: sessionToken,
      isActive: true,
      isRevoked: false,
      expiresAt: {
        gt: new Date(), // Not expired
      },
    },
    data: {
      lastActivityAt: new Date(),
    },
  });
}

/**
 * Revoke a specific session (hard delete)
 * Deletes session from DB and revokes refresh token in Redis
 * Returns whether it was the current session
 */
export async function revokeSession(sessionId: string, userId: string, currentSessionToken?: string | null) {
  const session = await prisma.session.findUnique({
    where: { id: sessionId },
    select: {
      id: true,
      userId: true,
      sessionToken: true,
      refreshToken: true,
    },
  });

  if (!session) {
    throw new Error("Session not found");
  }

  if (session.userId !== userId) {
    throw new Error("Unauthorized to revoke this session");
  }

  // Check if this is the current session
  const isCurrentSession = currentSessionToken && session.sessionToken === currentSessionToken;

  // Revoke refresh token in Redis if exists
  if (session.refreshToken) {
    try {
      await revokeRefreshTokenByJti(session.refreshToken);
    } catch (err) {
      // Ignore errors if token already revoked or expired
    }
  }

  // Hard delete session from DB
  await prisma.session.delete({
    where: { id: sessionId },
  });

  return { isCurrentSession };
}

/**
 * Revoke all sessions for a user (hard delete)
 * @param userId - User ID
 * @param currentSessionToken - Current session token to optionally exclude
 * @param includeCurrent - If true, includes current session in deletion (default: false)
 * @returns Object with deletedCount and wasCurrentIncluded
 */
export async function revokeAllUserSessions(
  userId: string,
  currentSessionToken: string,
  includeCurrent: boolean = false
) {
  // Build where clause
  const whereClause: any = {
    userId: userId,
  };

  // Exclude current session if not including it
  if (!includeCurrent) {
    whereClause.sessionToken = {
      not: currentSessionToken,
    };
  }

  // Get refresh tokens before deleting
  const sessionsToDelete = await prisma.session.findMany({
    where: whereClause,
    select: {
      refreshToken: true,
    },
  });

  // Revoke refresh tokens in Redis
  for (const session of sessionsToDelete) {
    if (session.refreshToken) {
      try {
        await revokeRefreshTokenByJti(session.refreshToken);
      } catch (err) {
        // Ignore errors if token already revoked
      }
    }
  }

  // Hard delete sessions
  const result = await prisma.session.deleteMany({
    where: whereClause,
  });

  return {
    deletedCount: result.count,
    wasCurrentIncluded: includeCurrent && result.count > 0,
  };
}

/**
 * Get device info from request for session creation
 */
export function getSessionDeviceInfo(req: Request) {
  const deviceInfo = extractDeviceInfo(req);
  return {
    ipAddress: deviceInfo.ipAddress,
    userAgent: deviceInfo.userAgent,
    location: null, // Can be enhanced with geo IP lookup later
  };
}

