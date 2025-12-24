import { Request, Response, NextFunction } from "express";
import { getDeviceLocationFromRequest, hasValidLocation } from "../middleware/deviceInfo.middleware";
import { 
  updateSessionLocation, 
  recordLocationHistory, 
  getLatestLocation,
  getLocationHistory,
  getChildrenLocations 
} from "../services/location.service";
import { BadRequestError, UnauthorizedError, ForbiddenError } from "../utils/errors";
import prisma from "../libs/prisma";

/**
 * POST /location/update
 * Update current session location and optionally record to history
 */
export async function updateLocation(req: Request, res: Response, next: NextFunction) {
  try {
    const userId = req.user?.id;
    const sessionToken = req.user?.jti;

    if (!userId || !sessionToken) {
      throw new UnauthorizedError("Authentication required");
    }

    const location = getDeviceLocationFromRequest(req);

    if (!hasValidLocation(location)) {
      throw new BadRequestError("Valid location data required (latitude and longitude)");
    }

    // Update session location
    await updateSessionLocation(sessionToken, location);

    // Optionally record to history (for child tracking)
    const recordHistory = req.body?.recordHistory === true;
    if (recordHistory) {
      await recordLocationHistory(userId, location);
    }

    res.json({
      success: true,
      message: "Location updated",
      data: {
        latitude: location.latitude,
        longitude: location.longitude,
        accuracy: location.accuracy,
        timestamp: new Date().toISOString(),
      },
    });
  } catch (error) {
    next(error);
  }
}

/**
 * GET /location/me
 * Get current user's latest location
 */
export async function getMyLocation(req: Request, res: Response, next: NextFunction) {
  try {
    const userId = req.user?.id;

    if (!userId) {
      throw new UnauthorizedError("Authentication required");
    }

    const location = await getLatestLocation(userId);

    res.json({
      success: true,
      data: location,
    });
  } catch (error) {
    next(error);
  }
}

/**
 * GET /location/history
 * Get current user's location history
 */
export async function getMyLocationHistory(req: Request, res: Response, next: NextFunction) {
  try {
    const userId = req.user?.id;

    if (!userId) {
      throw new UnauthorizedError("Authentication required");
    }

    const limit = Math.min(parseInt(req.query.limit as string) || 100, 500);
    const since = req.query.since ? new Date(req.query.since as string) : undefined;

    const history = await getLocationHistory(userId, limit, since);

    res.json({
      success: true,
      data: history,
      count: history.length,
    });
  } catch (error) {
    next(error);
  }
}

/**
 * GET /location/child/:childId
 * Get a linked child's latest location (parent only)
 */
export async function getChildLocation(req: Request, res: Response, next: NextFunction) {
  try {
    const parentId = req.user?.id;
    const { childId } = req.params;

    if (!parentId) {
      throw new UnauthorizedError("Authentication required");
    }

    // Verify parent-child link exists
    const link = await prisma.parentChildLink.findUnique({
      where: {
        parentId_childId: {
          parentId,
          childId,
        },
      },
    });

    if (!link) {
      throw new ForbiddenError("You are not linked to this child");
    }

    const location = await getLatestLocation(childId);

    res.json({
      success: true,
      data: location,
    });
  } catch (error) {
    next(error);
  }
}

/**
 * GET /location/child/:childId/history
 * Get a linked child's location history (parent only)
 */
export async function getChildLocationHistory(req: Request, res: Response, next: NextFunction) {
  try {
    const parentId = req.user?.id;
    const { childId } = req.params;

    if (!parentId) {
      throw new UnauthorizedError("Authentication required");
    }

    // Verify parent-child link exists
    const link = await prisma.parentChildLink.findUnique({
      where: {
        parentId_childId: {
          parentId,
          childId,
        },
      },
    });

    if (!link) {
      throw new ForbiddenError("You are not linked to this child");
    }

    const limit = Math.min(parseInt(req.query.limit as string) || 100, 500);
    const since = req.query.since ? new Date(req.query.since as string) : undefined;

    const history = await getLocationHistory(childId, limit, since);

    res.json({
      success: true,
      data: history,
      count: history.length,
    });
  } catch (error) {
    next(error);
  }
}

/**
 * GET /location/children
 * Get all linked children's latest locations (parent only)
 */
export async function getAllChildrenLocations(req: Request, res: Response, next: NextFunction) {
  try {
    const parentId = req.user?.id;

    if (!parentId) {
      throw new UnauthorizedError("Authentication required");
    }

    const children = await getChildrenLocations(parentId);

    res.json({
      success: true,
      data: children,
      count: children.length,
    });
  } catch (error) {
    next(error);
  }
}
