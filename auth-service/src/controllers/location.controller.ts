import { Request, Response, NextFunction } from "express";
import prisma from "../libs/prisma";
import { BadRequestError, NotFoundError, UnauthorizedError, ForbiddenError } from "../utils/errors";

/**
 * Update current user's location
 * POST /api/v1/location/update
 */
export const updateLocation = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const userId = req.user.id;
    const { latitude, longitude, accuracy, address } = req.body;

    if (latitude === undefined || longitude === undefined) {
      throw new BadRequestError("Latitude and longitude are required");
    }

    if (typeof latitude !== "number" || typeof longitude !== "number") {
      throw new BadRequestError("Latitude and longitude must be numbers");
    }

    // Validate coordinate ranges
    if (latitude < -90 || latitude > 90) {
      throw new BadRequestError("Latitude must be between -90 and 90");
    }
    if (longitude < -180 || longitude > 180) {
      throw new BadRequestError("Longitude must be between -180 and 180");
    }

    const location = await prisma.locationHistory.create({
      data: {
        userId,
        latitude,
        longitude,
        accuracy: accuracy || null,
        address: address || null,
      },
    });

    res.status(201).json({
      success: true,
      message: "Location updated successfully",
      location: {
        id: location.id,
        latitude: location.latitude,
        longitude: location.longitude,
        accuracy: location.accuracy,
        address: location.address,
        timestamp: location.timestamp,
      },
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Get current user's latest location
 * GET /api/v1/location/me
 */
export const getMyLocation = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const userId = req.user.id;

    const location = await prisma.locationHistory.findFirst({
      where: { userId },
      orderBy: { timestamp: "desc" },
    });

    if (!location) {
      return res.status(200).json({
        location: null,
        message: "No location data available",
      });
    }

    res.status(200).json({
      location: {
        id: location.id,
        latitude: location.latitude,
        longitude: location.longitude,
        accuracy: location.accuracy,
        address: location.address,
        timestamp: location.timestamp,
      },
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Get current user's location history
 * GET /api/v1/location/history
 */
export const getMyLocationHistory = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const userId = req.user.id;
    const { page = "1", limit = "20", from, to } = req.query;

    const pageNum = parseInt(page as string, 10);
    const limitNum = parseInt(limit as string, 10);

    if (pageNum < 1 || limitNum < 1 || limitNum > 100) {
      throw new BadRequestError("Invalid pagination parameters");
    }

    const skip = (pageNum - 1) * limitNum;

    // Build date filter
    const dateFilter: any = {};
    if (from) {
      dateFilter.gte = new Date(from as string);
    }
    if (to) {
      dateFilter.lte = new Date(to as string);
    }

    const whereClause: any = { userId };
    if (Object.keys(dateFilter).length > 0) {
      whereClause.timestamp = dateFilter;
    }

    const [locations, total] = await Promise.all([
      prisma.locationHistory.findMany({
        where: whereClause,
        orderBy: { timestamp: "desc" },
        skip,
        take: limitNum,
      }),
      prisma.locationHistory.count({ where: whereClause }),
    ]);

    const totalPages = Math.ceil(total / limitNum);

    res.status(200).json({
      data: locations.map((loc) => ({
        id: loc.id,
        latitude: loc.latitude,
        longitude: loc.longitude,
        accuracy: loc.accuracy,
        address: loc.address,
        timestamp: loc.timestamp,
      })),
      pagination: {
        page: pageNum,
        limit: limitNum,
        total,
        totalPages,
        hasNext: pageNum < totalPages,
        hasPrevious: pageNum > 1,
      },
    });
  } catch (err) {
    next(err);
  }
};


/**
 * Get all linked children's latest locations (parent only)
 * GET /api/v1/location/children
 */
export const getChildrenLocations = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const parentId = req.user.id;

    // Verify user is a parent
    if (req.user.role !== "PARENT") {
      throw new ForbiddenError("Only parents can access children's locations");
    }

    // Get all linked children
    const links = await prisma.parentChildLink.findMany({
      where: { parentId },
      include: {
        child: {
          select: {
            id: true,
            name: true,
            username: true,
            profileImg: true,
          },
        },
      },
    });

    if (links.length === 0) {
      return res.status(200).json({
        children: [],
        message: "No linked children found",
      });
    }

    // Get latest location for each child
    const childrenWithLocations = await Promise.all(
      links.map(async (link) => {
        const latestLocation = await prisma.locationHistory.findFirst({
          where: { userId: link.childId },
          orderBy: { timestamp: "desc" },
        });

        return {
          child: link.child,
          location: latestLocation
            ? {
                id: latestLocation.id,
                latitude: latestLocation.latitude,
                longitude: latestLocation.longitude,
                accuracy: latestLocation.accuracy,
                address: latestLocation.address,
                timestamp: latestLocation.timestamp,
              }
            : null,
        };
      })
    );

    res.status(200).json({
      children: childrenWithLocations,
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Get specific child's latest location (parent only)
 * GET /api/v1/location/child/:childId
 */
export const getChildLocation = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const parentId = req.user.id;
    const { childId } = req.params;

    if (!childId) {
      throw new BadRequestError("Child ID is required");
    }

    // Verify user is a parent
    if (req.user.role !== "PARENT") {
      throw new ForbiddenError("Only parents can access children's locations");
    }

    // Verify parent-child link exists
    const link = await prisma.parentChildLink.findFirst({
      where: { parentId, childId },
      include: {
        child: {
          select: {
            id: true,
            name: true,
            username: true,
            profileImg: true,
          },
        },
      },
    });

    if (!link) {
      throw new ForbiddenError("You are not linked to this child");
    }

    // Get latest location
    const location = await prisma.locationHistory.findFirst({
      where: { userId: childId },
      orderBy: { timestamp: "desc" },
    });

    res.status(200).json({
      child: link.child,
      location: location
        ? {
            id: location.id,
            latitude: location.latitude,
            longitude: location.longitude,
            accuracy: location.accuracy,
            address: location.address,
            timestamp: location.timestamp,
          }
        : null,
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Get specific child's location history (parent only)
 * GET /api/v1/location/child/:childId/history
 */
export const getChildLocationHistory = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const parentId = req.user.id;
    const { childId } = req.params;
    const { page = "1", limit = "20", from, to } = req.query;

    if (!childId) {
      throw new BadRequestError("Child ID is required");
    }

    // Verify user is a parent
    if (req.user.role !== "PARENT") {
      throw new ForbiddenError("Only parents can access children's locations");
    }

    // Verify parent-child link exists
    const link = await prisma.parentChildLink.findFirst({
      where: { parentId, childId },
      include: {
        child: {
          select: {
            id: true,
            name: true,
            username: true,
            profileImg: true,
          },
        },
      },
    });

    if (!link) {
      throw new ForbiddenError("You are not linked to this child");
    }

    const pageNum = parseInt(page as string, 10);
    const limitNum = parseInt(limit as string, 10);

    if (pageNum < 1 || limitNum < 1 || limitNum > 100) {
      throw new BadRequestError("Invalid pagination parameters");
    }

    const skip = (pageNum - 1) * limitNum;

    // Build date filter
    const dateFilter: any = {};
    if (from) {
      dateFilter.gte = new Date(from as string);
    }
    if (to) {
      dateFilter.lte = new Date(to as string);
    }

    const whereClause: any = { userId: childId };
    if (Object.keys(dateFilter).length > 0) {
      whereClause.timestamp = dateFilter;
    }

    const [locations, total] = await Promise.all([
      prisma.locationHistory.findMany({
        where: whereClause,
        orderBy: { timestamp: "desc" },
        skip,
        take: limitNum,
      }),
      prisma.locationHistory.count({ where: whereClause }),
    ]);

    const totalPages = Math.ceil(total / limitNum);

    res.status(200).json({
      child: link.child,
      data: locations.map((loc) => ({
        id: loc.id,
        latitude: loc.latitude,
        longitude: loc.longitude,
        accuracy: loc.accuracy,
        address: loc.address,
        timestamp: loc.timestamp,
      })),
      pagination: {
        page: pageNum,
        limit: limitNum,
        total,
        totalPages,
        hasNext: pageNum < totalPages,
        hasPrevious: pageNum > 1,
      },
    });
  } catch (err) {
    next(err);
  }
};
