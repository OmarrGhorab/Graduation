import express from "express";
import { authenticate } from "../middleware";
import {
  updateLocation,
  getMyLocation,
  getMyLocationHistory,
  getChildrenLocations,
  getChildLocation,
  getChildLocationHistory,
} from "../controllers/location.controller";

const router = express.Router();

// Current user's location endpoints
router.post("/", authenticate, updateLocation);
router.get("/me", authenticate, getMyLocation);
router.get("/history", authenticate, getMyLocationHistory);

// Parent endpoints for children's locations
router.get("/children", authenticate, getChildrenLocations);
router.get("/child/:childId", authenticate, getChildLocation);
router.get("/child/:childId/history", authenticate, getChildLocationHistory);

export default router;
