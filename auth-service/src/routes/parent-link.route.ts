import express from 'express';
import { authenticate } from '../middleware';
import {
  searchParents,
  sendParentLinkRequest,
  getPendingRequests,
  respondToRequest,
  getLinkedAccounts,
} from '../controllers/parent-link.controller';

const router = express.Router();

// All routes require authentication
router.get('/search', authenticate, searchParents);
router.post('/request', authenticate, sendParentLinkRequest);
router.get('/requests', authenticate, getPendingRequests);
router.post('/respond', authenticate, respondToRequest);
router.get('/linked', authenticate, getLinkedAccounts);

export default router;

