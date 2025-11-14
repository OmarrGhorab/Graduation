import express from 'express';
import { authenticate } from '../middleware';
import {
  searchParents,
  sendParentLinkRequest,
  getPendingRequests,
  respondToRequest,
  getLinkedAccounts,
  sendUnlinkRequest,
  getPendingUnlinkRequests,
  respondToUnlinkRequest,
} from '../controllers/parent-link.controller';

const router = express.Router();

// All routes require authentication
router.get('/search', authenticate, searchParents);
router.post('/request', authenticate, sendParentLinkRequest);
router.get('/requests', authenticate, getPendingRequests);
router.post('/respond', authenticate, respondToRequest);
router.get('/linked', authenticate, getLinkedAccounts);

// Unlink request routes
router.post('/unlink/request', authenticate, sendUnlinkRequest);
router.get('/unlink/requests', authenticate, getPendingUnlinkRequests);
router.post('/unlink/respond', authenticate, respondToUnlinkRequest);

export default router;

