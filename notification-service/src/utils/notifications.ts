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
 * Fetch a child's parents from auth-service
 */
export async function getChildParents(childId: string): Promise<any[]> {
  try {
    const response = await fetch(`${AUTH_SERVICE_URL}/api/v1/internal/users/${childId}/parents`, {
      headers: {
        "x-internal-service-secret": INTERNAL_SERVICE_SECRET,
      },
    });

    if (!response.ok) {
      console.warn(`[Notification] Failed to fetch parents for child ${childId}`);
      return [];
    }

    const body = await response.json();
    return body.data || [];
  } catch (error) {
    console.error(`[Notification] Error fetching parents for child ${childId}:`, error);
    return [];
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
    // Check for true duplicates - uses type-specific key fields, not just type alone
    const thirtySecondsAgo = new Date(Date.now() - 30 * 1000);
    const recentDuplicate = await prisma.notification.findFirst({
      where: { userId, type: data.type, createdAt: { gte: thirtySecondsAgo } },
      orderBy: { createdAt: 'desc' },
    });

    if (recentDuplicate) {
      const existing = recentDuplicate.data as Record<string, any>;

      // Security: deduplicate by same device
      if (data.newDevice && existing.newDevice &&
          data.newDevice.name === existing.newDevice?.name &&
          data.newDevice.platform === existing.newDevice?.platform) {
        console.log(`[Notification] Skipping duplicate security alert for user ${userId}`);
        return;
      }

      // Chat: only deduplicate if it's literally the same message
      if (data.type === 'chat.message') {
        if (data.message_id && existing.message_id && data.message_id === existing.message_id) {
          console.log(`[Notification] Skipping duplicate chat message ${data.message_id}`);
          return;
        }
        // Different message_id → allow through, don't deduplicate
      } else if (data.type.startsWith('LESSON_') || data.type.startsWith('CHILD_LESSON_')) {
        // Lesson events: deduplicate by lesson_id
        if (data.lesson_id && existing.lesson_id && data.lesson_id === existing.lesson_id) {
          console.log(`[Notification] Skipping duplicate lesson event ${data.lesson_id}`);
          return;
        }
      } else if (data.type.startsWith('ATTENDANCE_') || data.type.startsWith('CHILD_ATTENDANCE_')) {
        // Attendance: deduplicate by lesson_id + student
        if (data.lesson_id && existing.lesson_id && data.lesson_id === existing.lesson_id) {
          console.log(`[Notification] Skipping duplicate attendance event for lesson ${data.lesson_id}`);
          return;
        }
      } else if (data.type === 'parent_link_request' || data.type === 'unlink_request') {
        // Link requests: deduplicate by request_id
        if (data.requestId && existing.requestId && data.requestId === existing.requestId) {
          return;
        }
      } else if (!['SUBSCRIPTION_RENEWAL_SOON', 'CHILD_SUBSCRIPTION_RENEWAL_SOON', 'LESSON_REMINDER', 'CHILD_LESSON_REMINDER'].includes(data.type)) {
        // For all other non-repeating types, deduplicate by type within 30s
        console.log(`[Notification] Skipping duplicate ${data.type} for user ${userId}`);
        return;
      }
    }

    // Save to database
    const notification = await prisma.notification.create({
      data: { userId, type: data.type, data: data },
    });

    // Include DB notification id in the data so FCM and SSE share the same id
    const dataWithId = { ...data, notification_id: notification.id };

    // Enrich for UI delivery
    const enriched = enrichNotification({
      id: notification.id,
      type: data.type,
      data: dataWithId,
      read: false,
      createdAt: notification.createdAt,
    });

    // Always send via SSE for real-time in-app delivery
    const sseSent = sendToUser(userId, enriched);
    console.log(`[Notification] SSE delivery for user ${userId}: ${sseSent ? "sent" : "no active connections"}`);

    // Check user's notification preference before sending FCM
    const notificationsEnabled = await getUserNotificationPreference(userId);

    if (notificationsEnabled) {
      // Pass dataWithId so FCM payload includes the DB notification_id
      await sendFcmNotification(userId, dataWithId);
      console.log(`[Notification] FCM push sent for user ${userId} (notifications enabled)`);
    } else {
      console.log(`[Notification] Skipping FCM push for user ${userId} (notifications disabled)`);
    }

    const duration = Date.now() - startTime;
    console.log(`[Notification] Published notification for user ${userId}, type: ${data.type}, duration: ${duration}ms`);
  } catch (error) {
    const duration = Date.now() - startTime;
    console.error(`[Notification] Error publishing notification for user ${userId}, type: ${incomingData.type}, duration: ${duration}ms`, error);
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

    const actionInfo = data.action || getNotificationAction(data.type, data);

    // Prepare data payload (all values must be strings)
    const dataPayload: Record<string, string> = {
      type: data.type,
      ...Object.entries(data).reduce((acc, [key, value]) => {
        if (value !== undefined && value !== null && key !== 'action') {
          acc[key] = typeof value === "string" ? value : JSON.stringify(value);
        }
        return acc;
      }, {} as Record<string, string>),
    };

    if (actionInfo) {
      dataPayload.action = typeof actionInfo === "string" ? actionInfo : JSON.stringify(actionInfo);
    }

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
          const errorCode = resp.error?.code || "";
          
          if (token) {
            // Only mark as invalid if the token is actually expired or unregistered
            // We DON'T want to delete tokens if the error was a bad payload (our code error)
            const isTokenDead = errorCode === "messaging/registration-token-not-registered" || 
                               errorCode === "messaging/invalid-registration-token";

            if (isTokenDead) {
              invalidTokens.push(token);
              console.log(`[FCM] Marking token as invalid: ${token.substring(0, 20)}... (Error: ${errorCode})`);
            } else {
              console.error(
                `[FCM] Delivery failed to token ${token.substring(0, 20)}... (Non-token error: ${errorCode}):`,
                resp.error?.message || ""
              );
            }
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
    // Course & Lesson notifications
    "COURSE_ENROLLMENT": "New Student Enrolled",
    "LESSON_STARTED": "Lesson Started 🚀",
    "LESSON_CANCELED": "Lesson Canceled ⚠️",
    "LESSON_ENDED": "Lesson Ended ✅",
    "LESSON_RESCHEDULED": "Lesson Rescheduled 📅",
    "LESSON_REMINDER": "Upcoming Lesson 🔔",
    "CHILD_LESSON_REMINDER": "Child's Lesson Starting Soon 🔔",
    "CHILD_LESSON_STARTED": "Child's Lesson Started 🚀",
    "CHILD_LESSON_ENDED": "Child's Lesson Ended ✅",
    "ATTENDANCE_RECORDED": "Attendance Recorded ✅",
    "CHILD_ATTENDANCE_RECORDED": "Child's Attendance Recorded 📍",
    "ABSENCE_REQUEST_TEACHER": "New Absence Appeal 📋",
    "ABSENCE_REQUEST_PARENT": "Absence Update",
    "ATTENDANCE_STATUS_UPDATE": "Attendance Status Updated",
    "ATTENDANCE_FRAUD_TEACHER": "Attendance Fraud Alert ⚠️",
    "ATTENDANCE_FRAUD_PARENT": "Attendance Security Alert ⚠️",
    "VIDEO_READY": "Lesson Video Ready 🎬",
    "VIDEO_FAILED": "Video Processing Failed ❌",
    "PROGRESS_UPDATED": "Progress Updated 📈",
    "SUBSCRIPTION_RENEWAL_SOON": "Subscription Renewal Soon 💳",
    "CHILD_SUBSCRIPTION_RENEWAL_SOON": "Child's Subscription Renewing 💳",
    "SUBSCRIPTION_PAYMENT_FAILED": "Payment Failed ❌",
    "COURSE_REVIEW": "New Course Review ⭐",
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
    // Course & Lesson notifications
    case "COURSE_ENROLLMENT":
      return `${data.student_name || "A student"} has enrolled in your course: ${data.course_name || data.course_title || "Course"}`;
    case "LESSON_STARTED":
      return `The lesson "${data.lesson_title || "Lesson"}" has started! Get ready.`;
    case "LESSON_ENDED":
      return `The lesson "${data.lesson_title || "Lesson"}" in "${data.course_title || "your course"}" has ended. Attendance has been finalized.`;
    case "CHILD_LESSON_STARTED":
      return `Your child ${data.child_name || ""} started their lesson: "${data.lesson_title || "Lesson"}"`;
    case "CHILD_LESSON_ENDED":
      return `Your child ${data.child_name || ""} finished their lesson: "${data.lesson_title || "Lesson"}"`;
    case "LESSON_CANCELED":
      return `The lesson "${data.lesson_title || "Lesson"}" has been canceled.`;
    case "ATTENDANCE_RECORDED": {
      const statusMap: Record<string, string> = { PRESENT: "Present", LATE: "Late", ABSENT: "Absent", EXCUSED: "Excused" };
      const statusLabel = statusMap[data.status] || data.status || "recorded";
      return `You were marked ${statusLabel} for "${data.lesson_title || "Lesson"}" in "${data.course_title || "your course"}".`;
    }
    case "CHILD_ATTENDANCE_RECORDED": {
      const statusMap: Record<string, string> = { PRESENT: "Present", LATE: "Late", ABSENT: "Absent", EXCUSED: "Excused" };
      const statusLabel = statusMap[data.status] || data.status || "recorded";
      return `${data.child_name || "Your child"} was marked ${statusLabel} for "${data.lesson_title || "Lesson"}" in "${data.course_title || "a course"}".`;
    }
    case "ABSENCE_REQUEST_TEACHER":
      return data.body || `A student submitted an absence excuse for "${data.lesson_title || "a lesson"}". Tap to review and respond.`;
    case "ABSENCE_REQUEST_PARENT":
      return data.body || `An absence excuse for "${data.lesson_title || "a lesson"}" has been submitted for your child.`;
    case "ATTENDANCE_STATUS_UPDATE":
      return data.body || `Your attendance status has been updated for "${data.lesson_title || "a lesson"}".`;
    case "ATTENDANCE_FRAUD_TEACHER":
      return data.body || `A potential attendance fraud attempt was detected in "${data.course_title || "your course"}".`;
    case "ATTENDANCE_FRAUD_PARENT":
      return data.body || `An attendance issue was detected for your child in "${data.course_title || "a course"}".`;
    case "VIDEO_READY":
      return data.body || `The video for "${data.lesson_title || "your lesson"}" has been processed and is ready for students.`;
    case "VIDEO_FAILED":
      return data.body || `Video processing failed for "${data.lesson_title || "your lesson"}". Please re-upload.`;
    case "PROGRESS_UPDATED":
      return data.body || `Your progress has been updated. Overall progress: ${data.overall_progress ? Math.round(data.overall_progress) + "%" : "updated"}.`;
    case "LESSON_RESCHEDULED":
      return `The lesson "${data.lesson_title || "Lesson"}" has been moved to ${data.new_scheduled_at || "a new time"}.`;
    case "LESSON_REMINDER":
      return `"${data.lesson_title || "Lesson"}" starts in ${data.minutes_before || "a few"} minutes!`;
    case "CHILD_LESSON_REMINDER":
      return `${data.child_name || "Your child"} has a lesson "${data.lesson_title || ""}" starting in ${data.minutes_before || "a few"} minutes.`;

    case "SUBSCRIPTION_RENEWAL_SOON":
      return `Your subscription for "${data.course_name || "your course"}" is renewing in ${data.days_left || 3} days (${data.amount} ${data.currency}).`;
    case "CHILD_SUBSCRIPTION_RENEWAL_SOON":
      return `${data.child_name || "Your child"} has a subscription for "${data.course_name || "a course"}" renewing in ${data.days_left || 3} days (${data.amount} ${data.currency}).`;
    case "SUBSCRIPTION_PAYMENT_FAILED":
      return `We couldn't process your payment for "${data.course_name || "your course"}". Please check your payment method.`;
    case "COURSE_REVIEW":
      return `${data.student_name || "A student"} left a ${data.rating}-star review on "${data.course_name || data.course_title || "your course"}": "${data.review_text || ""}"`;
    default:
      return data.body || "You have a new notification";
  }
}

/**
 * Get notification image/icon based on type and data
 */
function getNotificationImage(
  type: string,
  data: Record<string, any>
): string | null {
  switch (type) {
    case "chat.message":
      return data.sender_image || null;
    case "parent_link_request":
    case "unlink_request":
      return data.child?.profileImg || null;
    case "parent_link_accepted":
    case "parent_link_request_accepted":
    case "parent_link_declined":
    case "parent_link_request_declined":
      return data.parent?.profileImg || null;
    // Security types get a specific icon or flag in UI usually
    case "security_new_device_blocked":
    case "security_device_verified":
    case "security_password_changed":
    case "security_account_locked":
      return "security-shield-icon";
    default:
      return null;
  }
}

/**
 * Get notification action (Deep link info)
 */
function getNotificationAction(
  type: string,
  data: Record<string, any>
): { type: string; target: string; params?: Record<string, any> } | null {
  switch (type) {
    case "chat.message":
      return {
        type: "navigate",
        target: "chat-detail",
        params: { conversationId: data.conversation_id },
      };
    case "parent_link_request":
      return {
        type: "navigate",
        target: "/link-requests",
        params: { requestId: data.requestId },
      };
    case "unlink_request":
      return {
        type: "navigate",
        target: "/unlink-requests",
        params: { requestId: data.requestId },
      };
    case "security_new_device_blocked":
      return {
        type: "navigate",
        target: "/security-settings",
      };
    case "LESSON_STARTED":
      return {
        type: "navigate",
        target: "/attendance-list",
        params: { lessonId: data.lesson_id },
      };
    case "COURSE_ENROLLMENT":
    case "LESSON_CANCELED":
    case "LESSON_RESCHEDULED":
    case "LESSON_REMINDER":
      return {
        type: "navigate",
        target: "/course-details",
        params: { id: data.course_id },
      };
    case "COURSE_REVIEW":
      return {
        type: "navigate",
        target: "/course-reviews",
        params: { id: data.course_id },
      };
    case "SUBSCRIPTION_RENEWAL_SOON":
    case "CHILD_SUBSCRIPTION_RENEWAL_SOON":
      return {
        type: "navigate",
        target: "/course-details",
        params: { id: data.course_id },
      };
    case "CHILD_LESSON_REMINDER":
      return {
        type: "navigate",
        target: "/course-details",
        params: { id: data.course_id },
      };
    case "CHILD_LESSON_STARTED":
    case "CHILD_LESSON_ENDED":
    case "CHILD_ATTENDANCE_RECORDED":
    case "ATTENDANCE_STATUS_UPDATE":
      return {
        type: "navigate",
        target: "/student-progress",
        params: { childId: data.child_id, courseId: data.course_id },
      };
    case "ATTENDANCE_RECORDED":
      return {
        type: "navigate",
        target: "/course-details",
        params: { id: data.course_id },
      };
    case "LESSON_ENDED":
      return {
        type: "navigate",
        target: "/course-details",
        params: { id: data.course_id },
      };
    case "ABSENCE_REQUEST_TEACHER":
      return {
        type: "navigate",
        target: "/absence-appeals",
        params: { lessonId: data.lesson_id, courseId: data.course_id },
      };
    case "ABSENCE_REQUEST_PARENT":
      return {
        type: "navigate",
        target: "/absence-history",
        params: {},
      };
    default:
      return null;
  }
}

/**
 * Enriches notification with UI-ready fields
 */
export function enrichNotification(notification: any) {
  const data = typeof notification.data === 'string'
    ? JSON.parse(notification.data)
    : (notification.data || {});

  return {
    id: notification.id,
    type: notification.type,
    title: getNotificationTitle(notification.type, data),
    body: getNotificationBody(notification.type, data),
    image: getNotificationImage(notification.type, data),
    action: getNotificationAction(notification.type, data),
    data: data, // Keep raw data for backward compatibility
    read: notification.read,
    createdAt: notification.createdAt,
  };
}

