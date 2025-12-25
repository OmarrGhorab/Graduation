import { Response } from "express";

// Store active SSE connections per user
const userConnections = new Map<string, Set<Response>>();

/**
 * Add an SSE connection for a user
 */
export function addConnection(userId: string, res: Response): void {
  if (!userConnections.has(userId)) {
    userConnections.set(userId, new Set());
  }
  userConnections.get(userId)!.add(res);
  console.log(`[SSE] User ${userId} connected. Total connections: ${userConnections.get(userId)!.size}`);
}

/**
 * Remove an SSE connection for a user
 */
export function removeConnection(userId: string, res: Response): void {
  const connections = userConnections.get(userId);
  if (connections) {
    connections.delete(res);
    console.log(`[SSE] User ${userId} disconnected. Remaining connections: ${connections.size}`);
    if (connections.size === 0) {
      userConnections.delete(userId);
    }
  }
}

/**
 * Send a real-time notification to all connected clients for a user
 */
export function sendToUser(userId: string, data: any): boolean {
  const connections = userConnections.get(userId);
  if (!connections || connections.size === 0) {
    console.log(`[SSE] No active connections for user ${userId}`);
    return false;
  }

  const message = `data: ${JSON.stringify(data)}\n\n`;
  let sentCount = 0;

  connections.forEach((res) => {
    try {
      res.write(message);
      sentCount++;
    } catch (error) {
      console.error(`[SSE] Error sending to user ${userId}:`, error);
      // Remove broken connection
      connections.delete(res);
    }
  });

  console.log(`[SSE] Sent notification to ${sentCount}/${connections.size} connections for user ${userId}`);
  return sentCount > 0;
}

/**
 * Get the number of active connections for a user
 */
export function getConnectionCount(userId: string): number {
  return userConnections.get(userId)?.size || 0;
}

/**
 * Get total active connections across all users
 */
export function getTotalConnections(): number {
  let total = 0;
  userConnections.forEach((connections) => {
    total += connections.size;
  });
  return total;
}
