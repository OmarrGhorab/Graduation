import { Request, Response, NextFunction } from "express";
import { UnauthorizedError, BadRequestError } from "../utils/errors";
import { AuthenticatedRequest } from "../middleware/auth";
import prisma from "../libs/prisma";
import {
  registerFcmToken,
  unregisterFcmToken,
} from "../utils/fcm-tokens";

/**
 * Get paginated notifications for the authenticated user
 * Supports pagination with default 10 notifications per page
 */
export const getNotifications = async (
  req: AuthenticatedRequest,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const userId = req.user!.id;
    const { page = "1", limit = "10", unreadOnly = "false" } = (req as any).query;

    const pageNum = parseInt(page as string, 10);
    const limitNum = parseInt(limit as string, 10);
    const unreadOnlyBool = unreadOnly === "true";

    if (pageNum < 1 || limitNum < 1 || limitNum > 50) {
      throw new BadRequestError("Invalid pagination parameters");
    }

    const skip = (pageNum - 1) * limitNum;

    // Build where clause
    const whereClause: any = { userId };
    if (unreadOnlyBool) {
      whereClause.read = false;
    }

    // Get notifications and total count in parallel
    const [notifications, total] = await Promise.all([
      prisma.notification.findMany({
        where: whereClause,
        select: {
          id: true,
          type: true,
          data: true,
          read: true,
          createdAt: true,
        },
        skip,
        take: limitNum,
        orderBy: {
          createdAt: "desc",
        },
      }),
      prisma.notification.count({
        where: whereClause,
      }),
    ]);

    const totalPages = Math.ceil(total / limitNum);

    res.status(200).json({
      data: notifications,
      pagination: {
        page: pageNum,
        limit: limitNum,
        total,
        totalPages,
        hasNext: pageNum < totalPages,
        hasPrevious: pageNum > 1,
      },
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Mark notifications as read
 * Can mark a single notification or all notifications as read
 */
export const markNotificationsRead = async (
  req: AuthenticatedRequest,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const userId = req.user!.id;
    const { notificationId, markAll = "false" } = (req as any).body;

    const markAllBool = markAll === "true";

    if (markAllBool) {
      // Mark all notifications as read for the user
      await prisma.notification.updateMany({
        where: {
          userId,
          read: false,
        },
        data: {
          read: true,
        },
      });

      res.status(200).json({
        message: "All notifications marked as read",
      });
    } else {
      // Mark specific notification as read
      if (!notificationId) {
        throw new BadRequestError("Notification ID is required when not marking all as read");
      }

      const notification = await prisma.notification.updateMany({
        where: {
          id: notificationId,
          userId,
          read: false,
        },
        data: {
          read: true,
        },
      });

      if (notification.count === 0) {
        throw new BadRequestError("Notification not found or already read");
      }

      res.status(200).json({
        message: "Notification marked as read",
      });
    }
  } catch (err) {
    next(err);
  }
};

/**
 * Register FCM token for push notifications
 * Mobile clients should call this after obtaining FCM token from Firebase
 */
export const registerFcmTokenEndpoint = async (
  req: AuthenticatedRequest,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const userId = req.user.id;
    const { token, deviceId, platform } = req.body;

    if (!token) {
      throw new BadRequestError("FCM token is required");
    }

    // Validate platform if provided
    if (platform && platform !== "ios" && platform !== "android") {
      throw new BadRequestError("Platform must be 'ios' or 'android'");
    }

    await registerFcmToken(userId, token, deviceId, platform);

    res.status(200).json({
      message: "FCM token registered successfully",
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Unregister FCM token
 * Mobile clients should call this on logout or when token becomes invalid
 */
export const unregisterFcmTokenEndpoint = async (
  req: AuthenticatedRequest,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const userId = req.user.id;
    const { token } = req.body;

    if (!token) {
      throw new BadRequestError("FCM token is required");
    }

    await unregisterFcmToken(userId, token);

    res.status(200).json({
      message: "FCM token unregistered successfully",
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Publish notification endpoint for other services to call
 * This is the main API that other microservices will use to send notifications
 */
export const publishNotificationEndpoint = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    const { userId, type, ...data } = req.body;

    if (!userId || !type) {
      throw new BadRequestError("userId and type are required");
    }

    // Import the publish function to avoid circular dependencies
    const { publishNotification } = await import("../utils/notifications");

    console.log(`[Notification Controller] Received publish request for user ${userId}, type: ${type}`, data);

    await publishNotification(userId, {
      type,
      ...data,
    });

    res.status(200).json({
      message: "Notification published successfully",
      data: {
        userId,
        type,
        timestamp: new Date().toISOString(),
      },
    });
  } catch (err) {
    next(err);
  }
};
