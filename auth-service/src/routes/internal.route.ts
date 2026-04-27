import express from 'express';
import { authenticateInternalService } from '../middleware';
import { getUserPreferencesInternal, validateTokenInternal, getBatchUsersInternal, getUserInternal, verifyAttendanceContextInternal, getParentsInternal, getChildrenInternal, searchUsersInternal } from '../controllers/internal.controller';

const router = express.Router();

// Internal endpoint to get user preferences (for notification-service)
router.get('/users/:userId/preferences', authenticateInternalService, getUserPreferencesInternal);

// Internal endpoint to get single user details (for courses-service)
router.get('/users/:userId', authenticateInternalService, getUserInternal);
router.get('/users/search', authenticateInternalService, searchUsersInternal);

// Internal endpoint to get batch user details (for chat-service)
router.post('/users/batch', authenticateInternalService, getBatchUsersInternal);

// Internal endpoint to validate tokens (for other services)
router.post('/validate-token', authenticateInternalService, validateTokenInternal);

// Internal endpoint for attendance verification (Requirement: 17.0)
router.post('/attendance/verify-context', authenticateInternalService, verifyAttendanceContextInternal);

// Internal endpoint to get parents of a child
router.get('/users/:userId/parents', authenticateInternalService, getParentsInternal);

// Internal endpoint to get children of a parent
router.get('/users/:userId/children', authenticateInternalService, getChildrenInternal);

export default router;
