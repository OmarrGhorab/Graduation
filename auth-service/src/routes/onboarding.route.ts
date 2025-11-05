import express from 'express';
import { authenticate } from '../middleware';
import { createOnboarding } from '../controllers/onboarding.controller';

const router = express.Router();

router.post("/", authenticate, createOnboarding);

export default router;