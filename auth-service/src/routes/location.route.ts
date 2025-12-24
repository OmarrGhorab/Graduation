import { Router } from "express";
import { authenticate } from "../middleware/auth.middleware";
import {
  updateLocation,
  getMyLocation,
  getMyLocationHistory,
  getChildLocation,
  getChildLocationHistory,
  getAllChildrenLocations,
} from "../controllers/location.controller";

const router = Router();

// All routes require authentication
router.use(authenticate);

// Current user location
router.post("/update", updateLocation);
router.get("/me", getMyLocation);
router.get("/history", getMyLocationHistory);

// Parent tracking endpoints
router.get("/children", getAllChildrenLocations);
router.get("/child/:childId", getChildLocation);
router.get("/child/:childId/history", getChildLocationHistory);

export default router;
