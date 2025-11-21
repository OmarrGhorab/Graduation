import redis from "../libs/redis";
import prisma from "../libs/prisma";

/**
 * Subscribe to Redis notifications for a user
 * Returns a Redis subscriber instance that can be used to listen for notifications
 */
export function createNotificationSubscriber(userId: string) {
  console.log(`Creating Redis subscriber for user: ${userId}`);
  
  const subscriber = redis.duplicate();
  
  subscriber.subscribe(`notifications:${userId}`, (err: Error | null | undefined) => {
    if (err) {
      console.error(`Error subscribing to notifications for user ${userId}:`, err);
    } else {
      console.log(`Subscribed to notifications for user ${userId}`);
    }
  });

  subscriber.on("error", (error: Error) => {
    console.error(`Redis subscriber error for user ${userId}:`, error);
  });

  console.log(`Redis subscriber created for user: ${userId}`);
  return subscriber;
}

/**
 * Publish a notification to a user's notification channel and save to database
 */
export async function publishNotification(
  userId: string,
  data: {
    type: string;
    [key: string]: any;
  }
): Promise<void> {
  try {
    // Publish to Redis for real-time delivery
    const channel = `notifications:${userId}`;
    const redisPromise = redis.publish(channel, JSON.stringify(data));

    // Save to database for persistence and pagination
    const dbPromise = prisma.notification.create({
      data: {
        userId,
        type: data.type,
        data: data,
      },
    });

    // Execute both in parallel, but don't await to keep it fast
    await Promise.all([redisPromise, dbPromise]);
  } catch (error) {
    console.error("Error publishing notification:", error);
    // Don't throw - notification failure shouldn't break the request
  }
}

/**
 * Get notification channel name for a user
 */
export function getNotificationChannel(userId: string): string {
  return `notifications:${userId}`;
}
