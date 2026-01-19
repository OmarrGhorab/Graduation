import { Request, Response, NextFunction } from "express";
import prisma from "../libs/prisma";

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
