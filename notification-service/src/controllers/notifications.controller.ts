import { Request, Response, NextFunction } from "express";
import { UnauthorizedError, BadRequestError } from "../utils/errors";
import { AuthenticatedRequest } from "../middleware/auth";
import prisma from "../libs/prisma";
import {
  registerFcmToken,
  unregisterFcmToken,
} from "../utils/fcm-tokens";
import { addConnection, removeConnection, getConnectionCount } from "../libs/sse";

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
 * Delete a specific notification
 */
export const deleteNotification = async (
  req: AuthenticatedRequest,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const userId = req.user.id;
    const { id } = req.params;

    if (!id) {
      throw new BadRequestError("Notification ID is required");
    }

    // Delete only if notification belongs to the user
    const result = await prisma.notification.deleteMany({
      where: {
        id,
        userId,
      },
    });

    if (result.count === 0) {
      throw new BadRequestError("Notification not found or already deleted");
    }

    res.status(200).json({
      message: "Notification deleted successfully",
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

/**
 * Update existing notifications endpoint
 * Used to update notification status (e.g., mark request as accepted/declined)
 */
export const updateNotificationsEndpoint = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    const { userId, type, matchCriteria, newType, dataUpdates } = req.body;

    if (!userId || !type || !matchCriteria) {
      throw new BadRequestError("userId, type, and matchCriteria are required");
    }

    const { updateNotificationsByType } = await import("../utils/notifications");

    console.log(`[Notification Controller] Updating notifications for user ${userId}, type: ${type}`, { matchCriteria, newType, dataUpdates });

    await updateNotificationsByType(userId, type, matchCriteria, {
      newType,
      dataUpdates,
    });

    res.status(200).json({
      message: "Notifications updated successfully",
    });
  } catch (err) {
    next(err);
  }
};

/**
 * SSE endpoint for real-time in-app notifications
 * Clients connect to this endpoint to receive notifications in real-time
 */
export const subscribeToNotifications = async (
  req: AuthenticatedRequest,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const userId = req.user.id;

    // Set SSE headers
    res.setHeader("Content-Type", "text/event-stream");
    res.setHeader("Cache-Control", "no-cache");
    res.setHeader("Connection", "keep-alive");
    res.setHeader("X-Accel-Buffering", "no"); // Disable nginx buffering
    res.flushHeaders();

    // Send initial connection confirmation
    res.write(`data: ${JSON.stringify({ type: "connected", message: "SSE connection established" })}\n\n`);

    // Add this connection to the user's connections
    addConnection(userId, res);

    // Send heartbeat every 30 seconds to keep connection alive
    const heartbeatInterval = setInterval(() => {
      try {
        res.write(`: heartbeat\n\n`);
      } catch (error) {
        clearInterval(heartbeatInterval);
      }
    }, 30000);

    // Handle client disconnect
    req.on("close", () => {
      clearInterval(heartbeatInterval);
      removeConnection(userId, res);
      console.log(`[SSE] Client disconnected for user ${userId}`);
    });

    // Log connection info
    console.log(`[SSE] New subscription for user ${userId}, total connections: ${getConnectionCount(userId)}`);
  } catch (err) {
    next(err);
  }
};
