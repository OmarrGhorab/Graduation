import redis from "../libs/redis";

/**
 * Subscribe to Redis notifications for a user
 * Returns a Redis subscriber instance that can be used to listen for notifications
 */
export function createNotificationSubscriber(userId: string) {
  const subscriber = redis.duplicate();
  
  subscriber.subscribe(`notifications:${userId}`, (err) => {
    if (err) {
      console.error(`Error subscribing to notifications for user ${userId}:`, err);
    } else {
      console.log(`Subscribed to notifications for user ${userId}`);
    }
  });

  return subscriber;
}

/**
 * Publish a notification to a user's notification channel
 */
export async function publishNotification(
  userId: string,
  data: {
    type: string;
    [key: string]: any;
  }
): Promise<void> {
  try {
    const channel = `notifications:${userId}`;
    await redis.publish(channel, JSON.stringify(data));
  } catch (error) {
    console.error("Error publishing notification to Redis:", error);
    // Don't throw - notification failure shouldn't break the request
  }
}

/**
 * Get notification channel name for a user
 */
export function getNotificationChannel(userId: string): string {
  return `notifications:${userId}`;
}

