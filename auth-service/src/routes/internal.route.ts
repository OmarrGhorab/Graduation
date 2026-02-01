import express from 'express';
import { authenticateInternalService } from '../middleware';
import { getUserPreferencesInternal, validateTokenInternal, getBatchUsersInternal } from '../controllers/internal.controller';

const router = express.Router();

// Internal endpoint to get user preferences (for notification-service)
router.get('/users/:userId/preferences', authenticateInternalService, getUserPreferencesInternal);

// Internal endpoint to get batch user details (for chat-service)
router.post('/users/batch', authenticateInternalService, getBatchUsersInternal);

// Internal endpoint to validate tokens (for other services)
router.post('/validate-token', authenticateInternalService, validateTokenInternal);

export default router;
