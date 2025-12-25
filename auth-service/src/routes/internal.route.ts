import express from 'express';
import { authenticateInternalService } from '../middleware';
import { getUserPreferencesInternal } from '../controllers/internal.controller';

const router = express.Router();

// Internal endpoint to get user preferences (for notification-service)
router.get('/users/:userId/preferences', authenticateInternalService, getUserPreferencesInternal);

export default router;
