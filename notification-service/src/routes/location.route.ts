import express from "express";
import { authenticate } from "../middleware/auth";
import { requestChildLocation } from "../controllers/location.controller";

const router = express.Router();

// Request child's location via silent push notification
// POST /api/v1/location/request/:childId
router.post("/request/:childId", authenticate, requestChildLocation);

export default router;
