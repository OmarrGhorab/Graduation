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
 * Revoke a specific session (soft delete)
 * Marks session as revoked in DB and revokes refresh token in Redis
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
      isRevoked: true,
    },
  });

  if (!session) {
    throw new Error("Session not found");
  }

  if (session.userId !== userId) {
    throw new Error("Unauthorized to revoke this session");
  }

  // Check if already revoked
  if (session.isRevoked) {
    // Still check if it's the current session for logout purposes
    const isCurrentSession = currentSessionToken && session.sessionToken === currentSessionToken;
    return { isCurrentSession };
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

  // Soft delete session - mark as revoked
  await prisma.session.update({
    where: { id: sessionId },
    data: {
      isRevoked: true,
      isActive: false,
      revokedAt: new Date(),
    },
  });

  return { isCurrentSession };
}

/**
 * Revoke all sessions for a user (soft delete)
 * Marks sessions as revoked in DB and revokes refresh tokens in Redis
 * @param userId - User ID
 * @param currentSessionToken - Current session token to optionally exclude
 * @param includeCurrent - If true, includes current session in revocation (default: false)
 * @returns Object with deletedCount and wasCurrentIncluded
 */
export async function revokeAllUserSessions(
  userId: string,
  currentSessionToken: string,
  includeCurrent: boolean = false
) {
  // Build where clause - only revoke sessions that aren't already revoked
  const whereClause: any = {
    userId: userId,
    isRevoked: false, // Only revoke sessions that aren't already revoked
  };

  // Exclude current session if not including it
  if (!includeCurrent) {
    whereClause.sessionToken = {
      not: currentSessionToken,
    };
  }

  // Get sessions to revoke (including refresh tokens)
  const sessionsToRevoke = await prisma.session.findMany({
    where: whereClause,
    select: {
      id: true,
      refreshToken: true,
    },
  });

  // Revoke refresh tokens in Redis
  for (const session of sessionsToRevoke) {
    if (session.refreshToken) {
      try {
        await revokeRefreshTokenByJti(session.refreshToken);
      } catch (err) {
        // Ignore errors if token already revoked
      }
    }
  }

  // Soft delete sessions - mark as revoked
  const result = await prisma.session.updateMany({
    where: whereClause,
    data: {
      isRevoked: true,
      isActive: false,
      revokedAt: new Date(),
    },
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

