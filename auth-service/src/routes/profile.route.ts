import express from 'express';
import { authenticate } from '../middleware/auth.middleware';
import {
  getProfile,
  updateProfile,
  uploadProfileImageEndpoint,
  checkUsernameAvailability,
} from '../controllers/profile.controller';

const router = express.Router();

// Check username availability (public endpoint - no auth required)
router.get('/check-username', checkUsernameAvailability);

// Get user profile
router.get('/', authenticate, getProfile);

// Update profile (name, username, password)
router.patch('/', authenticate, updateProfile);

// Upload/update profile image
router.post('/image', authenticate, uploadProfileImageEndpoint);

export default router;
