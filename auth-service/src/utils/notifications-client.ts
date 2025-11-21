import dotenv from "dotenv";

dotenv.config();

const NOTIFICATION_SERVICE_URL = process.env.NOTIFICATION_SERVICE_URL || "http://localhost:6003";

/**
 * Publish a notification via the notification service API
 * This replaces the direct Redis pub/sub approach
 */
export async function publishNotification(
  userId: string,
  data: {
    type: string;
    [key: string]: any;
  }
): Promise<void> {
  try {
    const response = await fetch(`${NOTIFICATION_SERVICE_URL}/api/v1/notifications/publish`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        userId,
        ...data,
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to publish notification: ${response.statusText}`);
    }

    console.log(`Notification published successfully for user ${userId}`);
  } catch (error) {
    console.error("Error publishing notification via service:", error);
    // Don't throw - notification failure shouldn't break the request
  }
}

/**
 * Get notification channel name for a user (kept for compatibility)
 * Note: This is no longer used for direct Redis access but kept for reference
 */
export function getNotificationChannel(userId: string): string {
  return `notifications:${userId}`;
}
