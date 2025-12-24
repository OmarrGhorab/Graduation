import prisma from "../libs/prisma";
import { messaging } from "../libs/firebase";
import admin from "../libs/firebase";
import { getUserFcmTokens, getUserFcmTokensWithPlatform } from "./fcm-tokens";

/**
 * Send a silent push notification to a specific device
 * Used for background data sync like location requests
 */
export async function sendSilentPushNotification(
  token: string,
  platform: string | null,
  data: Record<string, string>
): Promise<void> {
  if (!messaging) {
    console.warn("[FCM] FCM not available, skipping silent push notification");
    throw new Error("Push notifications are not configured");
  }

  try {
    const message: admin.messaging.Message = {
      token,
      data,
      // Android configuration for silent/data-only notification
      android: {
        priority: "high",
        // No notification field = silent/data-only message
      },
      // iOS configuration for silent notification
      apns: {
        payload: {
          aps: {
            "content-available": 1, // Silent notification flag
            // No alert, sound, or badge = silent
          },
        },
        headers: {
          "apns-priority": "10", // High priority for immediate delivery
          "apns-push-type": "background", // Background push type
        },
      },
    };

    const response = await messaging.send(message);
    console.log(`[FCM] Silent push notification sent successfully: ${response}`);
  } catch (error) {
    console.error("[FCM] Error sending silent push notification:", error);
    throw error;
  }
}

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
  const startTime = Date.now();
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

    const duration = Date.now() - startTime;
    console.log(`[Notification] Published notification for user ${userId}, type: ${data.type}, duration: ${duration}ms`);
  } catch (error) {
    const duration = Date.now() - startTime;
    console.error(`[Notification] Error publishing notification for user ${userId}, type: ${data.type}, duration: ${duration}ms`, error);
    if (error instanceof Error) {
      console.error(`[Notification] Error details: ${error.message}`, error.stack);
    }
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
    console.warn(`[FCM] FCM not available, skipping push notification for user ${userId}`);
    return;
  }

  try {
    // Get all FCM tokens with platform info for the user
    const tokensWithPlatform = await getUserFcmTokensWithPlatform(userId);

    if (tokensWithPlatform.length === 0) {
      console.log(`[FCM] No FCM tokens found for user ${userId}`);
      return;
    }

    // Separate tokens by platform for better targeting
    const iosTokens: string[] = [];
    const androidTokens: string[] = [];
    const unknownTokens: string[] = [];

    tokensWithPlatform.forEach(({ token, platform }) => {
      if (platform === "ios") {
        iosTokens.push(token);
      } else if (platform === "android") {
        androidTokens.push(token);
      } else {
        unknownTokens.push(token);
      }
    });

    const title = data.title || getNotificationTitle(data.type);
    const body = data.body || getNotificationBody(data.type, data);
    
    // Extract image URL if available (for profile images from child or parent)
    const imageUrl = data.child?.profileImg || data.parent?.profileImg || data.profileImg || data.imageUrl || null;

    // Prepare data payload (all values must be strings)
    const dataPayload: Record<string, string> = {
      type: data.type,
      ...Object.entries(data).reduce((acc, [key, value]) => {
        acc[key] = typeof value === "string" ? value : JSON.stringify(value);
        return acc;
      }, {} as Record<string, string>),
    };

    // Validate payload size (FCM limit: 4KB)
    const payloadSize = JSON.stringify(dataPayload).length;
    if (payloadSize > 4000) {
      console.warn(`[FCM] Payload size (${payloadSize} bytes) exceeds FCM limit (4KB) for user ${userId}`);
      // Truncate data payload if too large
      dataPayload.type = data.type;
      dataPayload.message = "Notification data too large";
    }

    const allTokens = [...iosTokens, ...androidTokens, ...unknownTokens];
    const promises: Promise<admin.messaging.BatchResponse>[] = [];
    const tokenGroups: string[][] = []; // Track which token groups were sent

    // Send to iOS devices with APNS-specific configuration
    if (iosTokens.length > 0) {
      const iosNotification: admin.messaging.MulticastMessage = {
        notification: {
          title,
          body,
          ...(imageUrl && { imageUrl }), // Add image if available
        },
        data: dataPayload,
        tokens: iosTokens,
        apns: {
          payload: {
            aps: {
              sound: "default",
              badge: 1,
              contentAvailable: true,
            },
          },
          headers: {
            "apns-priority": "10",
          },
          ...(imageUrl && {
            fcmOptions: {
              imageUrl,
            },
          }),
        },
      };
      promises.push(messaging.sendEachForMulticast(iosNotification));
      tokenGroups.push(iosTokens);
    }

    // Send to Android devices with Android-specific configuration
    if (androidTokens.length > 0) {
      const androidNotification: admin.messaging.MulticastMessage = {
        notification: {
          title,
          body,
          ...(imageUrl && { imageUrl }), // Add image if available
        },
        data: {
          ...dataPayload,
          click_action: "FLUTTER_NOTIFICATION_CLICK", // For deep linking
        },
        tokens: androidTokens,
        android: {
          priority: "high",
          notification: {
            channelId: "default",
            sound: "default",
            priority: "high" as const,
            ...(imageUrl && { imageUrl }), // Android-specific image
          },
        },
      };
      promises.push(messaging.sendEachForMulticast(androidNotification));
      tokenGroups.push(androidTokens);
    }

    // Send to unknown platform devices (fallback)
    if (unknownTokens.length > 0) {
      const unknownNotification: admin.messaging.MulticastMessage = {
        notification: {
          title,
          body,
          ...(imageUrl && { imageUrl }), // Add image if available
        },
        data: dataPayload,
        tokens: unknownTokens,
      };
      promises.push(messaging.sendEachForMulticast(unknownNotification));
      tokenGroups.push(unknownTokens);
    }

    // Send all notifications in parallel
    const responses = await Promise.all(promises);

    // Handle invalid tokens across all responses
    const invalidTokens: string[] = [];
    let totalSuccess = 0;
    let totalFailure = 0;

    responses.forEach((response, responseIdx) => {
      totalSuccess += response.successCount;
      totalFailure += response.failureCount;

      const tokenGroup = tokenGroups[responseIdx] || [];

      response.responses.forEach((resp, idx) => {
        if (!resp.success) {
          const token = tokenGroup[idx];
          if (token) {
            invalidTokens.push(token);
            console.error(
              `[FCM] Failed to send notification to token ${token.substring(0, 20)}...:`,
              resp.error?.code || "Unknown error",
              resp.error?.message || ""
            );
          }
        }
      });
    });

    // Remove invalid tokens from database
    if (invalidTokens.length > 0) {
      console.log(`[FCM] Removing ${invalidTokens.length} invalid tokens for user ${userId}`);
      await Promise.all(
        invalidTokens.map((token) =>
          prisma.fcmToken.deleteMany({ where: { token } }).catch((err) => {
            console.error(`[FCM] Error removing invalid token:`, err);
          })
        )
      );
    }

    console.log(
      `[FCM] Notification sent to ${totalSuccess}/${allTokens.length} devices for user ${userId} ` +
      `(iOS: ${iosTokens.length}, Android: ${androidTokens.length}, Unknown: ${unknownTokens.length})`
    );
  } catch (error) {
    console.error(`[FCM] Error sending FCM notification for user ${userId}:`, error);
    if (error instanceof Error) {
      console.error(`[FCM] Error details: ${error.message}`, error.stack);
    }
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
    parent_link_request_accepted: "Link Request Accepted",
    parent_link_request_declined: "Link Request Declined",
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
      return `${data.childName || data.child?.name || "A child"} wants to link with you`;
    case "parent_link_accepted":
    case "parent_link_request_accepted":
      return `${data.parentName || data.parent?.name || "A parent"} accepted your link request`;
    case "parent_link_declined":
    case "parent_link_request_declined":
      return `${data.parentName || data.parent?.name || "A parent"} declined your link request`;
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
