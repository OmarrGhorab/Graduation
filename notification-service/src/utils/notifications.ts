import prisma from "../libs/prisma";
import { messaging } from "../libs/firebase";
import admin from "../libs/firebase";
import { getUserFcmTokens } from "./fcm-tokens";

/**
 * Publish a notification to a user via FCM push notification and save to database
 */
export async function publishNotification(
  userId: string,
  data: {
    type: string;
    [key: string]: any;
  }
): Promise<void> {
  try {
    // Save to database for persistence and pagination
    const dbPromise = prisma.notification.create({
      data: {
        userId,
        type: data.type,
        data: data,
      },
    });

    // Send FCM push notification to all user's devices
    const fcmPromise = sendFcmNotification(userId, data);

    // Execute both in parallel
    await Promise.all([dbPromise, fcmPromise]);
  } catch (error) {
    console.error("Error publishing notification:", error);
    // Don't throw - notification failure shouldn't break the request
  }
}

/**
 * Send FCM push notification to all user's registered devices
 */
async function sendFcmNotification(
  userId: string,
  data: {
    type: string;
    [key: string]: any;
  }
): Promise<void> {
  // If Firebase is not initialized, skip FCM sending
  if (!messaging) {
    console.warn("FCM not available, skipping push notification");
    return;
  }

  try {
    // Get all FCM tokens for the user
    const tokens = await getUserFcmTokens(userId);

    if (tokens.length === 0) {
      console.log(`No FCM tokens found for user ${userId}`);
      return;
    }

    // Prepare notification payload
    const notification: admin.messaging.MulticastMessage = {
      notification: {
        title: getNotificationTitle(data.type),
        body: getNotificationBody(data.type, data),
      },
      data: {
        type: data.type,
        ...Object.entries(data).reduce((acc, [key, value]) => {
          acc[key] = typeof value === "string" ? value : JSON.stringify(value);
          return acc;
        }, {} as Record<string, string>),
      },
      tokens,
    };

    // Send to all devices
    const response = await messaging.sendEachForMulticast(notification);

    // Handle invalid tokens
    if (response.failureCount > 0) {
      const invalidTokens: string[] = [];
      response.responses.forEach((resp, idx) => {
        if (!resp.success) {
          invalidTokens.push(tokens[idx]);
          console.error(
            `Failed to send notification to token ${tokens[idx]}:`,
            resp.error
          );
        }
      });

      // Remove invalid tokens from database
      if (invalidTokens.length > 0) {
        await Promise.all(
          invalidTokens.map((token) =>
            prisma.fcmToken.deleteMany({ where: { token } }).catch(console.error)
          )
        );
      }
    }

    console.log(
      `FCM notification sent to ${response.successCount}/${tokens.length} devices for user ${userId}`
    );
  } catch (error) {
    console.error("Error sending FCM notification:", error);
    // Don't throw - FCM failure shouldn't break the request
  }
}

/**
 * Get notification title based on type
 */
function getNotificationTitle(type: string): string {
  const titles: Record<string, string> = {
    parent_link_request: "New Parent Link Request",
    parent_link_accepted: "Parent Link Accepted",
    parent_link_declined: "Parent Link Declined",
    unlink_request: "Unlink Request",
    unlink_request_accepted: "Unlink Request Accepted",
    unlink_request_declined: "Unlink Request Declined",
  };

  return titles[type] || "New Notification";
}

/**
 * Get notification body based on type and data
 */
function getNotificationBody(
  type: string,
  data: Record<string, any>
): string {
  switch (type) {
    case "parent_link_request":
      return `${data.childName || "A child"} wants to link with you`;
    case "parent_link_accepted":
      return `${data.parentName || "A parent"} accepted your link request`;
    case "parent_link_declined":
      return `${data.parentName || "A parent"} declined your link request`;
    case "unlink_request":
      return `${data.requesterName || "Someone"} wants to unlink from you`;
    case "unlink_request_accepted":
      return `${data.accepterName || "Someone"} accepted your unlink request`;
    case "unlink_request_declined":
      return `${data.declinerName || "Someone"} declined your unlink request`;
    default:
      return "You have a new notification";
  }
}
