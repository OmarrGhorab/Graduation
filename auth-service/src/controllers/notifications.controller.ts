import { Request, Response, NextFunction } from "express";
import { createNotificationSubscriber, getNotificationChannel } from "../utils/notifications";
import { UnauthorizedError } from "../utils/errors";

/**
 * Server-Sent Events (SSE) endpoint for real-time notifications
 * Clients can connect to this endpoint to receive real-time notifications
 */
export const streamNotifications = async (
  req: Request,
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
    res.setHeader("Access-Control-Allow-Origin", "http://localhost:3000");
    res.setHeader("Access-Control-Allow-Credentials", "true");

    // Create Redis subscriber for this user
    const subscriber = createNotificationSubscriber(userId);

    // Send initial connection message
    res.write(`data: ${JSON.stringify({ type: "connected", message: "Notification stream connected" })}\n\n`);

    // Listen for messages from Redis
    subscriber.on("message", (channel, message) => {
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
    req.on("close", () => {
      console.log(`Client disconnected from notification stream for user ${userId}`);
      clearInterval(heartbeatInterval);
      subscriber.unsubscribe();
      subscriber.quit();
      res.end();
    });

    // Handle errors
    subscriber.on("error", (error) => {
      console.error("Redis subscriber error:", error);
      res.write(`data: ${JSON.stringify({ type: "error", message: "Notification stream error" })}\n\n`);
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Get recent notifications (for polling fallback)
 * This is a simple endpoint that returns the last N notifications
 * For real-time, clients should use the SSE endpoint above
 */
export const getNotifications = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const userId = req.user.id;

    // For now, return empty array
    // In a full implementation, you might want to store notifications in a database
    // and return the most recent ones here
    res.status(200).json({
      data: [],
      message: "Use /api/v1/notifications/stream for real-time notifications",
    });
  } catch (err) {
    next(err);
  }
};

