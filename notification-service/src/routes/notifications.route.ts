import express from 'express';
import { authenticate } from '../middleware/auth';
import { authenticateInternalService } from '../middleware/internal-auth';
import {
  streamNotifications,
  getNotifications,
  markNotificationsRead,
  publishNotificationEndpoint,
} from '../controllers/notifications.controller';

const router = express.Router();

// Internal endpoint for other services to publish notifications
// Protected by internal service authentication
router.post('/publish', authenticateInternalService, publishNotificationEndpoint);

// SSE endpoint for real-time notifications (requires user authentication)
router.get('/stream', authenticate, streamNotifications);

// Polling endpoint (fallback) (requires user authentication)
router.get('/', authenticate, getNotifications);

// Mark notifications as read (requires user authentication)
router.patch('/read', authenticate, markNotificationsRead);

export default router;
