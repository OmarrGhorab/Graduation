import express from 'express';
import { authenticate } from '../middleware/auth';
import { authenticateInternalService } from '../middleware/internal-auth';
import {
  getNotifications,
  markNotificationsRead,
  publishNotificationEndpoint,
  registerFcmTokenEndpoint,
  unregisterFcmTokenEndpoint,
} from '../controllers/notifications.controller';

const router = express.Router();

// Internal endpoint for other services to publish notifications
// Protected by internal service authentication
router.post('/publish', authenticateInternalService, publishNotificationEndpoint);

// FCM token management endpoints (requires user authentication)
router.post('/register-token', authenticate, registerFcmTokenEndpoint);
router.delete('/unregister-token', authenticate, unregisterFcmTokenEndpoint);

// Get notification history (requires user authentication)
router.get('/', authenticate, getNotifications);

// Mark notifications as read (requires user authentication)
router.patch('/read', authenticate, markNotificationsRead);

export default router;
