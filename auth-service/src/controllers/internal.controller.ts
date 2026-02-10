import { Request, Response, NextFunction } from "express";
import prisma from "../libs/prisma";
import { verifyAccessToken } from "../utils/tokens";
import { UnauthorizedError, BadRequestError } from "../utils/errors";

/**
 * Get user preferences for internal service calls
 * Used by notification-service to check if user has notifications enabled
 */
export const getUserPreferencesInternal = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    const { userId } = req.params;

    if (!userId) {
      return res.status(400).json({ error: "userId is required" });
    }

    const preferences = await prisma.userPreference.findUnique({
      where: { userId },
      select: {
        notifications: true,
        language: true,
        themePreference: true,
      },
    });

    // Return default values if no preferences exist
    res.status(200).json({
      notifications: preferences?.notifications ?? true,
      language: preferences?.language ?? "en",
      themePreference: preferences?.themePreference ?? "light",
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Internal endpoint to validate a token and return user info
 * Requirement: 16.1, 16.2
 */
export const validateTokenInternal = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    const { token } = req.body;

    if (!token) {
      throw new BadRequestError("Token is required");
    }

    try {
      const payload = await verifyAccessToken(token);

      if (payload.type !== "access") {
        throw new UnauthorizedError("Invalid token type");
      }

      // Check if session exists and is active
      const session = await prisma.session.findFirst({
        where: {
          sessionToken: payload.jti,
          userId: payload.sub,
          isRevoked: false,
          expiresAt: {
            gt: new Date(),
          },
        },
      });

      if (!session) {
        return res.status(401).json({
          valid: false,
          error: "Session not found or expired",
        });
      }

      res.status(200).json({
        valid: true,
        userId: payload.sub,
        role: payload.role,
        jti: payload.jti,
      });
    } catch (err) {
      res.status(401).json({
        valid: false,
        error: "Invalid or expired token",
      });
    }
  } catch (err) {
    next(err);
  }
};

/**
 * Get batch user details for internal services
 * Used by chat-service to enrich messages
 */
export const getBatchUsersInternal = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    const { userIds } = req.body;

    if (!Array.isArray(userIds)) {
      throw new BadRequestError("userIds must be an array");
    }

    if (userIds.length === 0) {
      return res.status(200).json({});
    }

    const users = await prisma.user.findMany({
      where: {
        id: {
          in: userIds,
        },
      },
      select: {
        id: true,
        name: true,
        profileImg: true,
        role: true,
      },
    });

    // Convert to map for easier lookup
    const userMap = users.reduce((acc, user) => {
      acc[user.id] = {
        id: user.id,
        name: user.name,
        image: user.profileImg,
        role: user.role,
      };
      return acc;
    }, {} as Record<string, any>);

    res.status(200).json(userMap);
  } catch (err) {
    next(err);
  }
};

/**
 * Get single user details for internal services
 * Used by courses-service to get teacher info
 */
export const getUserInternal = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    const { userId } = req.params;

    if (!userId) {
      throw new BadRequestError("userId is required");
    }

    const user = await prisma.user.findUnique({
      where: { id: userId },
      select: {
        id: true,
        name: true,
        username: true,
        email: true,
        profileImg: true,
        role: true,
      },
    });

    if (!user) {
      return res.status(404).json({
        success: false,
        error: "User not found",
      });
    }

    res.status(200).json({
      success: true,
      data: {
        id: user.id,
        name: user.name,
        username: user.username,
        email: user.email,
        profileImg: user.profileImg,
        role: user.role,
      },
    });
  } catch (err) {
    next(err);
  }
};
