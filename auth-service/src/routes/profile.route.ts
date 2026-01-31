import express from 'express';
import { authenticate } from '../middleware/auth.middleware';
import {
  getProfile,
  updateProfile,
  uploadProfileImageEndpoint,
  checkUsernameAvailability,
  updatePreferences,
  getPreferences,
} from '../controllers/profile.controller';

const router = express.Router();

// Get user profile
router.get('/', authenticate, getProfile);

// Update profile (name, username, password)
router.put('/', authenticate, updateProfile);

// Check username availability (public endpoint - no auth required)
router.get('/username/check', checkUsernameAvailability);

// Upload/update profile image
router.post('/image', authenticate, uploadProfileImageEndpoint);

// Get user preferences
router.get('/preferences', authenticate, getPreferences);

// Update user preferences (theme, language, notifications, newsletter)
router.patch('/preferences', authenticate, updatePreferences);

export default router;
