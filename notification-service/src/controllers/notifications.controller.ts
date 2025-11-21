import { Request, Response, NextFunction } from "express";
import { createNotificationSubscriber, getNotificationChannel } from "../utils/notifications";
import { UnauthorizedError, BadRequestError } from "../utils/errors";
import { AuthenticatedRequest } from "../middleware/auth";
import prisma from "../libs/prisma";

/**
 * Server-Sent Events (SSE) endpoint for real-time notifications
 * Clients can connect to this endpoint to receive real-time notifications
 */
export const streamNotifications = async (
  req: AuthenticatedRequest,
  res: Response,
  next: NextFunction
) => {
  try {
    console.log("SSE endpoint hit, checking authentication...");
    
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const userId = req.user.id;
    console.log(`User authenticated: ${userId}, setting up SSE...`);

    // Set SSE headers BEFORE any other operations
    res.setHeader("Content-Type", "text/event-stream");
    res.setHeader("Cache-Control", "no-cache");
    res.setHeader("Connection", "keep-alive");
    res.setHeader("Access-Control-Allow-Origin", "*");
    res.setHeader("Access-Control-Allow-Credentials", "false");

    console.log("SSE headers set, creating Redis subscriber...");

    // Create Redis subscriber for this user
    const subscriber = createNotificationSubscriber(userId);
    
    console.log("Redis subscriber created, sending initial connection message...");

    // Send initial connection message
    res.write(`data: ${JSON.stringify({ type: "connected", message: "Notification stream connected" })}\n\n`);

    console.log("Initial message sent, setting up Redis message listener...");

    // Listen for messages from Redis
    subscriber.on("message", (channel: string, message: string) => {
      try {
        const data = JSON.parse(message);
        res.write(`data: ${JSON.stringify(data)}\n\n`);
      } catch (error) {
        console.error("Error parsing notification message:", error);
      }
    });

    // Send heartbeat every 30 seconds to keep connection alive
    const heartbeatInterval = setInterval(() => {
      res.write(`data: ${JSON.stringify({ type: "heartbeat", timestamp: new Date().toISOString() })}\n\n`);
    }, 30000);

    // Handle client disconnect
    (req as any).on("close", () => {
      console.log(`Client disconnected from notification stream for user ${userId}`);
      clearInterval(heartbeatInterval);
      subscriber.unsubscribe();
      subscriber.quit();
      res.end();
    });

    // Log connection details for debugging
    console.log(`Active SSE connections for user ${userId}: ${getActiveConnections(userId)}`);

    // Handle errors
    subscriber.on("error", (error: Error) => {
      console.error("Redis subscriber error:", error);
      res.write(`data: ${JSON.stringify({ type: "error", message: "Notification stream error" })}\n\n`);
    });

  } catch (err) {
    next(err);
  }
};

// Track active connections per user for debugging
const activeConnections = new Map<string, number>();

function getActiveConnections(userId: string): string {
  const current = activeConnections.get(userId) || 0;
  activeConnections.set(userId, current + 1);
  return `${current + 1} connections`;
}

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
