import { Request } from "express";
import prisma from "../libs/prisma";
import { revokeRefreshTokenByJti } from "./tokens";
import { getDeviceInfoFromRequest, DeviceInfo } from "../middleware/deviceInfo.middleware";

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

export interface SessionActivity {
  id: string;
  action: string;
  ipAddress: string | null;
  userAgent: string | null;
  location: string | null;
  timestamp: Date;
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
 * Removes session from DB and revokes refresh token in Redis
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
  const isCurrentSession =
    !!currentSessionToken && session.sessionToken === currentSessionToken;

  // Revoke refresh token in Redis if exists
  if (session.refreshToken) {
    try {
      await revokeRefreshTokenByJti(session.refreshToken);
    } catch (err) {
      // Ignore errors if token already revoked or expired
    }
  }

  // Hard delete session from database
  await prisma.session.delete({
    where: { id: sessionId },
  });

  return { isCurrentSession };
}

/**
 * Revoke all sessions for a user (hard delete)
 * Removes sessions from DB and revokes refresh tokens in Redis
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

  // Get sessions to revoke (including refresh tokens)
  const sessionsToRevoke = await prisma.session.findMany({
    where: whereClause,
    select: {
      id: true,
      refreshToken: true,
      sessionToken: true,
    },
  });

  // Check if current session will be included
  const wasCurrentIncluded = sessionsToRevoke.some(
    (s) => s.sessionToken === currentSessionToken
  );

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

  // Hard delete sessions from database
  const result = await prisma.session.deleteMany({
    where: whereClause,
  });

  return {
    deletedCount: result.count,
    wasCurrentIncluded,
  };
}

/**
 * Get detailed session information including device and activity
 */
export async function getSessionDetails(sessionId: string, userId: string) {
  const session = await prisma.session.findUnique({
    where: { id: sessionId },
    include: {
      device: {
        select: {
          id: true,
          deviceName: true,
          platform: true,
          ipAddress: true,
          userAgent: true,
          isTrusted: true,
          lastLoginAt: true,
          createdAt: true,
        },
      },
    },
  });

  if (!session) {
    throw new Error("Session not found");
  }

  if (session.userId !== userId) {
    throw new Error("Unauthorized to view this session");
  }

  const isExpired = new Date(session.expiresAt) < new Date();
  const isActive = session.isActive && !session.isRevoked && !isExpired;

  // Parse user agent for readable device info
  const deviceInfo = parseUserAgent(session.userAgent || session.device?.userAgent);

  return {
    id: session.id,
    // Device info
    device: {
      id: session.device?.id || null,
      name: session.device?.deviceName || deviceInfo.deviceName,
      platform: session.device?.platform || deviceInfo.platform,
      browser: deviceInfo.browser,
      os: deviceInfo.os,
      isTrusted: session.device?.isTrusted || false,
    },
    // Network info
    network: {
      ipAddress: session.ipAddress || session.device?.ipAddress || null,
      location: session.location || null,
      userAgent: session.userAgent || session.device?.userAgent || null,
    },
    // Session state
    status: {
      isActive,
      isRevoked: session.isRevoked,
      isExpired,
    },
    // Timestamps
    timestamps: {
      createdAt: session.createdAt,
      lastActivityAt: session.lastActivityAt,
      expiresAt: session.expiresAt,
      revokedAt: session.revokedAt,
      deviceLastLoginAt: session.device?.lastLoginAt || null,
    },
  };
}

/**
 * Parse user agent string to extract device info
 */
function parseUserAgent(userAgent: string | null | undefined): {
  deviceName: string;
  platform: string | null;
  browser: string | null;
  os: string | null;
} {
  if (!userAgent) {
    return {
      deviceName: "Unknown Device",
      platform: null,
      browser: null,
      os: null,
    };
  }

  let browser: string | null = null;
  let os: string | null = null;
  let platform: string | null = null;

  // Detect browser
  if (userAgent.includes("Chrome") && !userAgent.includes("Edg")) {
    browser = "Chrome";
  } else if (userAgent.includes("Safari") && !userAgent.includes("Chrome")) {
    browser = "Safari";
  } else if (userAgent.includes("Firefox")) {
    browser = "Firefox";
  } else if (userAgent.includes("Edg")) {
    browser = "Edge";
  } else if (userAgent.includes("Opera") || userAgent.includes("OPR")) {
    browser = "Opera";
  }

  // Detect OS
  if (userAgent.includes("Windows")) {
    os = "Windows";
    platform = "WEB";
  } else if (userAgent.includes("Mac OS")) {
    os = "macOS";
    platform = "WEB";
  } else if (userAgent.includes("Linux") && !userAgent.includes("Android")) {
    os = "Linux";
    platform = "WEB";
  } else if (userAgent.includes("Android")) {
    os = "Android";
    platform = "ANDROID";
  } else if (userAgent.includes("iPhone") || userAgent.includes("iPad")) {
    os = "iOS";
    platform = "IOS";
  }

  // Build device name
  const parts = [browser, os].filter(Boolean);
  const deviceName = parts.length > 0 ? `${parts.join(" on ")}` : "Unknown Device";

  return { deviceName, platform, browser, os };
}

/**
 * Clean up expired sessions for a user (hard delete)
 */
export async function cleanupExpiredSessions(userId: string) {
  const result = await prisma.session.deleteMany({
    where: {
      userId,
      expiresAt: {
        lt: new Date(),
      },
    },
  });

  return result.count;
}

/**
 * Get location data from IP address using free IP geolocation service
 */
async function getLocationFromIp(ip: string): Promise<string | null> {
  // Override removed to use real IP

  if (!ip || ip === '127.0.0.1' || ip === '::1' || ip.startsWith('192.168.') || ip.startsWith('10.') || ip.startsWith('172.')) {
    return 'Local/Private Network';
  }

  try {
    // Use ip-api.com - no auth required, no bot protection, server-friendly
    const response = await fetch(`http://ip-api.com/json/${ip}?fields=status,message,country,region,city`);

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    const data = await response.json();

    console.log('Location API response:', data);

    if (data.status === 'success' && data.city && data.country) {
      const parts = [data.city, data.region, data.country].filter(Boolean);
      const location = parts.length > 0 ? parts.join(', ') : null;
      console.log('Parsed location:', location);
      return location;
    } else {
      console.log('Location API failed:', data.message || 'unknown error');
      return null;
    }
  } catch (error) {
    console.error('Failed to get location from IP:', error);
    return null;
  }
}

/**
 * Get device info from request for session creation
 * Uses device info from headers (set by middleware)
 * Falls back to IP geolocation if no location header provided
 */
export async function getSessionDeviceInfo(req: Request): Promise<{
  ipAddress: string;
  userAgent: string | null;
  location: string | null;
  deviceName: string;
  platform: string;
  timezone: string | null;
  appVersion: string | null;
  deviceModel: string | null;
  osVersion: string | null;
}> {
  const deviceInfo = getDeviceInfoFromRequest(req);
  
  // Use client-provided location if available, otherwise try IP geolocation
  let location = deviceInfo.location;
  if (!location && deviceInfo.ipAddress) {
    location = await getLocationFromIp(deviceInfo.ipAddress);
  }

  return {
    ipAddress: deviceInfo.ipAddress,
    userAgent: deviceInfo.userAgent,
    location: location,
    deviceName: deviceInfo.deviceName,
    platform: deviceInfo.platform,
    timezone: deviceInfo.timezone,
    appVersion: deviceInfo.appVersion,
    deviceModel: deviceInfo.deviceModel,
    osVersion: deviceInfo.osVersion,
  };
}

