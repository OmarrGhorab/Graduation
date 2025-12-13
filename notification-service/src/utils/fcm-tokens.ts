import prisma from "../libs/prisma";
import { BadRequestError } from "./errors";

/**
 * Register or update an FCM token for a user
 * If the token already exists for the user, it will be updated
 * If the token exists for another user, it will be transferred to the current user
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
    } else {
      // Token belongs to another user, transfer it to current user
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
 * Unregister all FCM tokens for a user (useful for logout or account deletion)
 */
export async function unregisterAllUserFcmTokens(userId: string): Promise<void> {
  await prisma.fcmToken.deleteMany({
    where: { userId },
  });
}

