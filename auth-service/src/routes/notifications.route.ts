import express from 'express';
import { authenticate } from '../middleware';
import {
  streamNotifications,
  getNotifications,
} from '../controllers/notifications.controller';

const router = express.Router();

// SSE endpoint for real-time notifications
router.get('/stream', authenticate, streamNotifications);

// Polling endpoint (fallback)
router.get('/', authenticate, getNotifications);

export default router;

