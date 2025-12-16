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
  const startTime = Date.now();
  try {
    if (!process.env.INTERNAL_SERVICE_SECRET) {
      console.warn(`[Notification Client] INTERNAL_SERVICE_SECRET not configured, notification may fail for user ${userId}`);
    }

    const response = await fetch(`${NOTIFICATION_SERVICE_URL}/api/v1/notifications/publish`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(process.env.INTERNAL_SERVICE_SECRET && {
          'x-internal-secret': process.env.INTERNAL_SERVICE_SECRET,
        }),
      },
      body: JSON.stringify({
        userId,
        ...data,
      }),
    });

    if (!response.ok) {
      const errorText = await response.text().catch(() => response.statusText);
      throw new Error(`Failed to publish notification: ${response.status} ${errorText}`);
    }

    const duration = Date.now() - startTime;
    console.log(`[Notification Client] Published notification for user ${userId}, type: ${data.type}, duration: ${duration}ms`);
  } catch (error) {
    const duration = Date.now() - startTime;
    console.error(`[Notification Client] Error publishing notification for user ${userId}, type: ${data.type}, duration: ${duration}ms`, error);
    if (error instanceof Error) {
      console.error(`[Notification Client] Error details: ${error.message}`, error.stack);
    }
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
