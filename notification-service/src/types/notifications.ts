/**
 * TypeScript types for notification data structures
 * These types ensure consistency between backend and frontend (React Native)
 */

/**
 * Base notification type
 */
export interface BaseNotification {
  id: string;
  userId: string;
  type: NotificationType;
  data: NotificationData;
  read: boolean;
  createdAt: string; // ISO 8601 date string
}

/**
 * Notification types supported by the system
 */
export type NotificationType =
  | "parent_link_request"
  | "parent_link_accepted"
  | "parent_link_declined"
  | "parent_link_request_accepted"
  | "parent_link_request_declined"
  | "unlink_request"
  | "unlink_request_accepted"
  | "unlink_request_declined";

/**
 * Union type for all notification data structures
 */
export type NotificationData =
  | ParentLinkRequestData
  | ParentLinkResponseData
  | UnlinkRequestData
  | UnlinkResponseData;

/**
 * Data structure for parent link request notifications
 */
export interface ParentLinkRequestData {
  type: "parent_link_request";
  requestId: string;
  child?: {
    id: string;
    username: string;
    name: string;
  };
  parent?: {
    id: string;
    username: string;
    name: string;
  };
  childName?: string;
  parentName?: string;
  status?: "PENDING" | "ACCEPTED" | "DECLINED";
  createdAt?: string;
}

/**
 * Data structure for parent link response notifications
 */
export interface ParentLinkResponseData {
  type: "parent_link_accepted" | "parent_link_declined" | "parent_link_request_accepted" | "parent_link_request_declined";
  requestId: string;
  parent?: {
    id: string;
    username: string;
    name: string;
  };
  child?: {
    id: string;
    username: string;
    name: string;
  };
  parentName?: string;
  childName?: string;
  status: "ACCEPTED" | "DECLINED";
  respondedAt?: string;
}

/**
 * Data structure for unlink request notifications
 */
export interface UnlinkRequestData {
  type: "unlink_request";
  requestId: string;
  requester?: {
    id: string;
    username: string;
    name: string;
  };
  requesterName?: string;
  status?: "PENDING" | "ACCEPTED" | "DECLINED";
  createdAt?: string;
}

/**
 * Data structure for unlink response notifications
 */
export interface UnlinkResponseData {
  type: "unlink_request_accepted" | "unlink_request_declined";
  requestId: string;
  accepter?: {
    id: string;
    username: string;
    name: string;
  };
  decliner?: {
    id: string;
    username: string;
    name: string;
  };
  accepterName?: string;
  declinerName?: string;
  status: "ACCEPTED" | "DECLINED";
  respondedAt?: string;
}

/**
 * Paginated notification response
 */
export interface PaginatedNotificationsResponse {
  data: BaseNotification[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
    hasNext: boolean;
    hasPrevious: boolean;
  };
}

/**
 * Register FCM token request
 */
export interface RegisterFcmTokenRequest {
  token: string;
  deviceId?: string;
  platform?: "ios" | "android";
}

/**
 * Unregister FCM token request
 */
export interface UnregisterFcmTokenRequest {
  token: string;
}

/**
 * Mark notification as read request
 */
export interface MarkNotificationReadRequest {
  notificationId?: string;
  markAll?: boolean;
}

/**
 * Type guard to check if notification data is a parent link request
 */
export function isParentLinkRequestData(data: NotificationData): data is ParentLinkRequestData {
  return data.type === "parent_link_request";
}

/**
 * Type guard to check if notification data is a parent link response
 */
export function isParentLinkResponseData(data: NotificationData): data is ParentLinkResponseData {
  return (
    data.type === "parent_link_accepted" ||
    data.type === "parent_link_declined" ||
    data.type === "parent_link_request_accepted" ||
    data.type === "parent_link_request_declined"
  );
}

/**
 * Type guard to check if notification data is an unlink request
 */
export function isUnlinkRequestData(data: NotificationData): data is UnlinkRequestData {
  return data.type === "unlink_request";
}

/**
 * Type guard to check if notification data is an unlink response
 */
export function isUnlinkResponseData(data: NotificationData): data is UnlinkResponseData {
  return data.type === "unlink_request_accepted" || data.type === "unlink_request_declined";
}

