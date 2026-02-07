import prisma from "../libs/prisma";
import { messaging } from "../libs/firebase";
import admin from "../libs/firebase";
import { getUserFcmTokens, getUserFcmTokensWithPlatform } from "./fcm-tokens";
import { sendToUser } from "../libs/sse";

// Auth service URL for fetching user preferences
const AUTH_SERVICE_URL = process.env.AUTH_SERVICE_URL || "http://localhost:6001";
const INTERNAL_SERVICE_SECRET = process.env.INTERNAL_SERVICE_SECRET || "";

/**
 * Fetch user notification preference from auth-service
 * Returns true if notifications are enabled, false otherwise
 */
async function getUserNotificationPreference(userId: string): Promise<boolean> {
  try {
    const response = await fetch(`${AUTH_SERVICE_URL}/api/v1/internal/users/${userId}/preferences`, {
      headers: {
        "x-internal-service-secret": INTERNAL_SERVICE_SECRET,
      },
    });

    if (!response.ok) {
      console.warn(`[Notification] Failed to fetch user preferences for ${userId}, defaulting to enabled`);
      return true; // Default to enabled if we can't fetch
    }

    const data = await response.json();
    // notifications field: true = enabled, false = disabled, null/undefined = default to true
    return data.notifications !== false;
  } catch (error) {
    console.error(`[Notification] Error fetching user preferences for ${userId}:`, error);
    return true; // Default to enabled on error
  }
}

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
 * Update an existing notification's data (e.g., mark request as accepted/declined)
 * Also sends real-time update via SSE
 */
export async function updateNotification(
  notificationId: string,
  updates: {
    type?: string;
    data?: Record<string, any>;
  }
): Promise<void> {
  try {
    const notification = await prisma.notification.findUnique({
      where: { id: notificationId },
    });

    if (!notification) {
      console.warn(`[Notification] Notification ${notificationId} not found for update`);
      return;
    }

    // Merge existing data with updates
    const existingData = notification.data as Record<string, any>;
    const newData = updates.data ? { ...existingData, ...updates.data } : existingData;

    const updated = await prisma.notification.update({
      where: { id: notificationId },
      data: {
        type: updates.type || notification.type,
        data: newData,
      },
    });

    // Send real-time update via SSE
    const payload = {
      id: updated.id,
      type: updated.type,
      data: newData,
      read: updated.read,
      createdAt: updated.createdAt.toISOString(),
      updated: true, // Flag to indicate this is an update, not a new notification
    };

    sendToUser(notification.userId, payload);
    console.log(`[Notification] Updated notification ${notificationId} for user ${notification.userId}`);
  } catch (error) {
    console.error(`[Notification] Error updating notification ${notificationId}:`, error);
  }
}

/**
 * Update notifications by type and data criteria
 * Useful for updating request notifications when they're accepted/declined
 */
export async function updateNotificationsByType(
  userId: string,
  type: string,
  matchCriteria: Record<string, any>,
  updates: {
    newType?: string;
    dataUpdates?: Record<string, any>;
  }
): Promise<void> {
  try {
    // Find notifications matching the criteria
    const notifications = await prisma.notification.findMany({
      where: {
        userId,
        type,
      },
    });

    for (const notification of notifications) {
      const data = notification.data as Record<string, any>;

      // Check if notification matches the criteria
      const matches = Object.entries(matchCriteria).every(([key, value]) => {
        // Handle nested keys like "child.id"
        const keys = key.split('.');
        let current = data;
        for (const k of keys) {
          if (current && typeof current === 'object' && k in current) {
            current = current[k];
          } else {
            return false;
          }
        }
        return current === value;
      });

      if (matches) {
        await updateNotification(notification.id, {
          type: updates.newType,
          data: updates.dataUpdates,
        });
      }
    }
  } catch (error) {
    console.error(`[Notification] Error updating notifications by type:`, error);
  }
}

/**
 * Publish a notification to a user
 * - Checks for recent duplicates to prevent spam
 * - Always saves to database
 * - Always sends via SSE for real-time in-app delivery
 * - Only sends FCM push if user's notification preference is enabled
 */
export async function publishNotification(
  userId: string,
  incomingData: {
    type: string;
    [key: string]: any;
  }
): Promise<void> {
  const startTime = Date.now();
  try {
    // Flatten data if it contains a nested 'data' property (common from other services like chat-service)
    let data = { ...incomingData };
    if (data.data && typeof data.data === 'object' && !Array.isArray(data.data)) {
      const nested = data.data;
      delete data.data;
      data = { ...data, ...nested };
    }
    // Check for duplicate notification in the last 30 seconds
    // This prevents duplicate notifications from rapid retries or double-clicks
    const thirtySecondsAgo = new Date(Date.now() - 30 * 1000);
    const recentDuplicate = await prisma.notification.findFirst({
      where: {
        userId,
        type: data.type,
        createdAt: { gte: thirtySecondsAgo },
      },
      orderBy: { createdAt: 'desc' },
    });

    if (recentDuplicate) {
      // Check if the notification data is similar (same device for security alerts)
      const existingData = recentDuplicate.data as Record<string, any>;
      const isSameDevice = data.newDevice && existingData.newDevice &&
        data.newDevice.name === existingData.newDevice?.name &&
        data.newDevice.platform === existingData.newDevice?.platform;

      if (isSameDevice || data.type === recentDuplicate.type) {
        console.log(`[Notification] Skipping duplicate notification for user ${userId}, type: ${data.type} (sent ${Math.round((Date.now() - recentDuplicate.createdAt.getTime()) / 1000)}s ago)`);
        return; // Skip duplicate
      }
    }

    // Save to database for persistence and pagination
    const notification = await prisma.notification.create({
      data: {
        userId,
        type: data.type,
        data: data,
      },
    });

    // Prepare notification payload for real-time delivery
    const notificationPayload = {
      id: notification.id,
      type: data.type,
      data: data,
      read: false,
      createdAt: notification.createdAt.toISOString(),
    };

    // Always send via SSE for real-time in-app delivery
    const sseSent = sendToUser(userId, notificationPayload);
    console.log(`[Notification] SSE delivery for user ${userId}: ${sseSent ? "sent" : "no active connections"}`);

    // Check user's notification preference before sending FCM
    const notificationsEnabled = await getUserNotificationPreference(userId);

    if (notificationsEnabled) {
      // User has notifications enabled - send FCM push
      await sendFcmNotification(userId, data);
      console.log(`[Notification] FCM push sent for user ${userId} (notifications enabled)`);
    } else {
      console.log(`[Notification] Skipping FCM push for user ${userId} (notifications disabled)`);
    }

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

    const title = data.title || getNotificationTitle(data.type, data);
    const body = data.body || getNotificationBody(data.type, data);

    // Extract image URL if available
    // For chat messages, use sender_image; for other notifications, use child/parent profile images
    const imageUrl = data.sender_image || data.child?.profileImg || data.parent?.profileImg || data.profileImg || data.imageUrl || null;

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
              badge: parseInt(data.unread_count) || 1,
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
function getNotificationTitle(type: string, data?: Record<string, any>): string {
  // For chat messages, use conversation name as title
  if (type === "chat.message" && data) {
    return data.conversation_name || data.sender_name || "New Message";
  }

  const titles: Record<string, string> = {
    parent_link_request: "New Parent Link Request",
    parent_link_accepted: "Parent Link Accepted",
    parent_link_declined: "Parent Link Declined",
    parent_link_request_accepted: "Link Request Accepted",
    parent_link_request_declined: "Link Request Declined",
    unlink_request: "Unlink Request",
    unlink_request_accepted: "Unlink Request Accepted",
    unlink_request_declined: "Unlink Request Declined",
    // Security notifications
    security_new_device_blocked: "Security Alert: New Device Login Attempt",
    security_device_verified: "New Device Added",
    security_password_changed: "Password Changed",
    security_account_locked: "Account Locked",
    // Chat notifications
    "chat.message": "New Message",
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
    // Security notifications
    case "security_new_device_blocked":
      return data.body || `Someone tried to log in from a new device (${data.newDevice?.name || "Unknown"}). If this wasn't you, please secure your account.`;
    case "security_device_verified":
      return `A new device (${data.deviceName || "Unknown"}) has been added to your account.`;
    case "security_password_changed":
      return "Your password was recently changed. If you didn't do this, please contact support immediately.";
    case "security_account_locked":
      return "Your account has been locked due to suspicious activity. Please verify your identity to unlock.";
    // Chat notifications
    case "chat.message":
      const senderName = data.sender_name || "Someone";
      const body = data.body || data.content || "sent you a message";

      // If it's a group chat, show "Sender: Message"
      // In chat-service, for direct chats, conversation_name is set to sender_name
      if (data.conversation_name && data.conversation_name !== data.sender_name) {
        return `${senderName}: ${body}`;
      }

      // For direct chats, conversation title is already the sender name, so just show body
      return body;
    default:
      return "You have a new notification";
  }
}
