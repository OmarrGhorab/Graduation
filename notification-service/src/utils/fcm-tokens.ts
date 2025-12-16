import prisma from "../libs/prisma";
import { BadRequestError } from "./errors";

/**
 * Register or update an FCM token for a user
 * If the token already exists for the user, it will be updated
 * If the token exists for another user, it will be transferred to the current user
 * (This handles cases where user logs in on a device previously used by another user)
 */
export async function registerFcmToken(
  userId: string,
  token: string,
  deviceId?: string,
  platform?: "ios" | "android"
): Promise<void> {
  if (!token || token.trim().length === 0) {
    throw new BadRequestError("FCM token is required");
  }

  // Validate token format (basic check - FCM tokens are typically long strings)
  if (token.length < 50) {
    throw new BadRequestError("Invalid FCM token format");
  }

  // Validate platform if provided
  if (platform && platform !== "ios" && platform !== "android") {
    throw new BadRequestError("Platform must be 'ios' or 'android'");
  }

  try {
    // Check if token already exists
    const existingToken = await prisma.fcmToken.findUnique({
      where: { token },
    });

    if (existingToken) {
      if (existingToken.userId === userId) {
        // Token already registered for this user, just update metadata
        await prisma.fcmToken.update({
          where: { token },
          data: {
            deviceId: deviceId || existingToken.deviceId,
            platform: platform || existingToken.platform,
            updatedAt: new Date(),
          },
        });
        console.log(`[FCM Token] Updated token for user ${userId}, platform: ${platform || existingToken.platform}`);
      } else {
        // Token belongs to another user - transfer it (device reused by different user)
        console.log(`[FCM Token] Transferring token from user ${existingToken.userId} to user ${userId}`);
        await prisma.fcmToken.update({
          where: { token },
          data: {
            userId,
            deviceId: deviceId || existingToken.deviceId,
            platform: platform || existingToken.platform,
            updatedAt: new Date(),
          },
        });
      }
    } else {
      // New token, create it
      await prisma.fcmToken.create({
        data: {
          userId,
          token,
          deviceId: deviceId || null,
          platform: platform || null,
        },
      });
      console.log(`[FCM Token] Registered new token for user ${userId}, platform: ${platform || "unknown"}`);
    }
  } catch (error) {
    console.error(`[FCM Token] Error registering token for user ${userId}:`, error);
    throw error;
  }
}

/**
 * Unregister an FCM token for a user
 */
export async function unregisterFcmToken(
  userId: string,
  token: string
): Promise<void> {
  if (!token || token.trim().length === 0) {
    throw new BadRequestError("FCM token is required");
  }

  // Only delete if the token belongs to this user
  await prisma.fcmToken.deleteMany({
    where: {
      userId,
      token,
    },
  });
}

/**
 * Get all FCM tokens for a user
 */
export async function getUserFcmTokens(userId: string): Promise<string[]> {
  const tokens = await prisma.fcmToken.findMany({
    where: { userId },
    select: { token: true },
  });

  return tokens.map((t: { token: string }) => t.token);
}

/**
 * Get FCM tokens with platform information for a user
 */
export async function getUserFcmTokensWithPlatform(userId: string): Promise<Array<{ token: string; platform: string | null }>> {
  const tokens = await prisma.fcmToken.findMany({
    where: { userId },
    select: { token: true, platform: true },
  });

  return tokens.map((t) => ({ token: t.token, platform: t.platform }));
}

/**
 * Unregister all FCM tokens for a user (useful for logout or account deletion)
 */
export async function unregisterAllUserFcmTokens(userId: string): Promise<void> {
  await prisma.fcmToken.deleteMany({
    where: { userId },
  });
}

