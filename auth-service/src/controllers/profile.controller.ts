import { Request, Response, NextFunction } from "express";
import prisma from "../libs/prisma";
import { BadRequestError, NotFoundError } from "../utils/errors";
import { uploadProfileImage, deleteImageFromCloudinary } from "../utils/cloudinaryUpload";
import { generateUniqueUsername } from "../utils/username";
import bcrypt from "bcrypt";

/**
 * Upload or update profile image
 * Removes existing Cloudinary image if present
 */
export const uploadProfileImageEndpoint = async (req: Request, res: Response, next: NextFunction) => {
  try {
    if (!req.user) {
      throw new BadRequestError("User not authenticated");
    }

    const userId = req.user.id;
    const { profileImg } = req.body;

    if (!profileImg) {
      throw new BadRequestError("Profile image is required");
    }

    // Fetch current user to check for existing image
    const currentUser = await prisma.user.findUnique({
      where: { id: userId },
      select: { profileImg: true },
    });

    if (!currentUser) {
      throw new NotFoundError("User not found");
    }

    let newImageUrl: string;

    // Check if new image is a URL or base64 data
    if (profileImg.startsWith('http')) {
      // For URLs (e.g., from Google), store directly
      newImageUrl = profileImg;
    } else {
      // For base64 data, upload to Cloudinary
      newImageUrl = await uploadProfileImage(profileImg, userId);
    }

    // Delete old Cloudinary image if it exists
    if (currentUser.profileImg?.includes("cloudinary.com")) {
      try {
        await deleteImageFromCloudinary(currentUser.profileImg);
      } catch (error) {
        console.error("Failed to delete old profile image:", error);
        // Continue even if deletion fails
      }
    }

    // Update user profile image
    const updatedUser = await prisma.user.update({
      where: { id: userId },
      data: { profileImg: newImageUrl },
      select: {
        id: true,
        username: true,
        name: true,
        email: true,
        profileImg: true,
      },
    });

    res.status(200).json({
      message: "Profile image updated successfully",
      user: updatedUser,
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Update user profile (name, username, password)
 * Username can only be changed once every 2 weeks
 */
export const updateProfile = async (req: Request, res: Response, next: NextFunction) => {
  try {
    if (!req.user) {
      throw new BadRequestError("User not authenticated");
    }

    const userId = req.user.id;
    const { name, username, password, currentPassword, bio, goals, interests } = req.body;

    // Fetch current user
    const currentUser = await prisma.user.findUnique({
      where: { id: userId },
      select: {
        id: true,
        name: true,
        username: true,
        email: true,
        profileImg: true,
        password: true,
        lastUsernameChange: true,
      },
    });

    if (!currentUser) {
      throw new NotFoundError("User not found");
    }

    const updateData: any = {};

    // Update name if provided
    if (name !== undefined && name.trim() !== "") {
      updateData.name = name.trim();
    }

    // Update username if provided
    if (username !== undefined && username.trim() !== "") {
      const newUsername = username.trim().toLowerCase();

      // Check if username is different
      if (newUsername !== currentUser.username) {
        // Check if user can change username (2 weeks cooldown)
        const twoWeeksAgo = new Date();
        twoWeeksAgo.setDate(twoWeeksAgo.getDate() - 14);

        if (currentUser.lastUsernameChange && currentUser.lastUsernameChange > twoWeeksAgo) {
          const nextChangeDate = new Date(currentUser.lastUsernameChange);
          nextChangeDate.setDate(nextChangeDate.getDate() + 14);

          throw new BadRequestError(
            `You can only change your username once every 2 weeks. Next change available on ${nextChangeDate.toLocaleDateString()}`
          );
        }

        // Check if username is already taken
        const existingUser = await prisma.user.findUnique({
          where: { username: newUsername },
          select: { id: true },
        });

        if (existingUser) {
          // Generate suggestions
          const suggestions = await generateUsernameSuggestions(newUsername);
          throw new BadRequestError(
            `Username "${newUsername}" is already taken. Try: ${suggestions.join(", ")}`
          );
        }

        updateData.username = newUsername;
        updateData.lastUsernameChange = new Date();
      }
    }

    // Update password if provided
    if (password !== undefined && password.trim() !== "") {
      if (!currentPassword) {
        throw new BadRequestError("Current password is required to change password");
      }

      // Check if user has a password (OAuth users don't have passwords)
      if (!currentUser.password) {
        throw new BadRequestError("Cannot change password for OAuth accounts. Please set a password first.");
      }

      // Verify current password
      const isPasswordValid = await bcrypt.compare(currentPassword, currentUser.password);
      if (!isPasswordValid) {
        throw new BadRequestError("Current password is incorrect");
      }

      // Validate new password
      if (password.length < 6) {
        throw new BadRequestError("Password must be at least 6 characters long");
      }

      // Hash new password
      const hashedPassword = await bcrypt.hash(password, 10);
      updateData.password = hashedPassword;
    }

    // Update bio if provided
    if (bio !== undefined) {
      if (bio.trim().length > 200) {
        throw new BadRequestError("Bio must be at most 200 characters");
      }
      updateData.bio = bio.trim();
    }

    // Update goals if provided
    if (goals !== undefined) {
      if (!Array.isArray(goals)) {
        throw new BadRequestError("Goals must be an array");
      }
      if (goals.length > 3) {
        throw new BadRequestError("Maximum 3 goals allowed");
      }
      updateData.goals = goals;
    }

    // Update interests if provided
    if (interests !== undefined && Array.isArray(interests)) {
      // This will be handled separately after the user update
      // to properly manage the Interest and UserInterest tables
    }

    // Update user
    const updatedUser = await prisma.user.update({
      where: { id: userId },
      data: updateData,
      select: {
        id: true,
        username: true,
        name: true,
        email: true,
        profileImg: true,
        lastUsernameChange: true,
        bio: true,
        goals: true,
      },
    });

    // Handle interests update if provided
    if (interests !== undefined && Array.isArray(interests)) {
      // Remove existing interests
      await prisma.userInterest.deleteMany({
        where: { userId },
      });

      // Add new interests
      for (const interestName of interests) {
        if (!interestName || typeof interestName !== "string") {
          continue;
        }

        // Find or create the interest
        const interest = await prisma.interest.upsert({
          where: { name: interestName.trim() },
          create: { name: interestName.trim() },
          update: {},
        });

        // Link user to interest
        await prisma.userInterest.create({
          data: {
            userId,
            interestId: interest.id,
          },
        });
      }
    }

    // Fetch updated user with interests
    const userWithInterests = await prisma.user.findUnique({
      where: { id: userId },
      select: {
        id: true,
        username: true,
        name: true,
        email: true,
        profileImg: true,
        lastUsernameChange: true,
        bio: true,
        goals: true,
        interests: {
          include: {
            interest: true,
          },
        },
      },
    });

    res.status(200).json({
      message: "Profile updated successfully",
      user: {
        ...userWithInterests,
        interests: userWithInterests?.interests.map((ui) => ({
          id: ui.interest.id,
          name: ui.interest.name,
        })),
      },
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Generate username suggestions when username is taken
 */
async function generateUsernameSuggestions(baseUsername: string): Promise<string[]> {
  const suggestions: string[] = [];
  
  // Try with numbers
  for (let i = 1; i <= 3; i++) {
    const suggestion = `${baseUsername}${Math.floor(Math.random() * 1000)}`;
    const exists = await prisma.user.findUnique({
      where: { username: suggestion },
      select: { id: true },
    });
    if (!exists) {
      suggestions.push(suggestion);
    }
  }

  // Try with underscores and numbers
  if (suggestions.length < 3) {
    const suggestion = `${baseUsername}_${Math.floor(Math.random() * 100)}`;
    const exists = await prisma.user.findUnique({
      where: { username: suggestion },
      select: { id: true },
    });
    if (!exists) {
      suggestions.push(suggestion);
    }
  }

  return suggestions.slice(0, 3);
}

/**
 * Get user profile
 */
export const getProfile = async (req: Request, res: Response, next: NextFunction) => {
  try {
    if (!req.user) {
      throw new BadRequestError("User not authenticated");
    }

    const userId = req.user.id;

    const user = await prisma.user.findUnique({
      where: { id: userId },
      select: {
        id: true,
        username: true,
        name: true,
        email: true,
        profileImg: true,
        dateOfBirth: true,
        gender: true,
        country: true,
        role: true,
        bio: true,
        goals: true,
        newsletterEnabled: true,
        lastUsernameChange: true,
        createdAt: true,
        interests: {
          include: {
            interest: true,
          },
        },
        preferences: {
          select: {
            language: true,
            themePreference: true,
            notifications: true,
          },
        },
      },
    });

    if (!user) {
      throw new NotFoundError("User not found");
    }

    // Calculate if user can change username
    const twoWeeksAgo = new Date();
    twoWeeksAgo.setDate(twoWeeksAgo.getDate() - 14);
    const canChangeUsername = !user.lastUsernameChange || user.lastUsernameChange <= twoWeeksAgo;

    let nextUsernameChangeDate = null;
    if (!canChangeUsername && user.lastUsernameChange) {
      nextUsernameChangeDate = new Date(user.lastUsernameChange);
      nextUsernameChangeDate.setDate(nextUsernameChangeDate.getDate() + 14);
    }

    res.status(200).json({
      user: {
        ...user,
        interests: user.interests.map((ui) => ({
          id: ui.interest.id,
          name: ui.interest.name,
        })),
      },
      canChangeUsername,
      nextUsernameChangeDate,
    });
  } catch (err) {
    next(err);
  }
};


/**
 * Check if username is available
 * Public endpoint - no authentication required
 */
export const checkUsernameAvailability = async (req: Request, res: Response, next: NextFunction) => {
  try {
    const { username } = req.query;

    if (!username || typeof username !== "string") {
      throw new BadRequestError("Username is required");
    }

    const normalizedUsername = username.trim().toLowerCase();

    if (normalizedUsername.length < 3) {
      return res.status(400).json({
        available: false,
        message: "Username must be at least 3 characters long",
      });
    }

    if (normalizedUsername.length > 30) {
      return res.status(400).json({
        available: false,
        message: "Username must be at most 30 characters long",
      });
    }

    // Check if username contains only valid characters (alphanumeric, underscore, hyphen)
    const validUsernameRegex = /^[a-z0-9_-]+$/;
    if (!validUsernameRegex.test(normalizedUsername)) {
      return res.status(400).json({
        available: false,
        message: "Username can only contain letters, numbers, underscores, and hyphens",
      });
    }

    // Check if username exists
    const existingUser = await prisma.user.findUnique({
      where: { username: normalizedUsername },
      select: { id: true },
    });

    if (existingUser) {
      // Generate suggestions
      const suggestions = await generateUsernameSuggestions(normalizedUsername);
      
      return res.status(200).json({
        available: false,
        message: "Username is already taken",
        suggestions: suggestions.length > 0 ? suggestions : undefined,
      });
    }

    res.status(200).json({
      available: true,
      message: "Username is available",
      username: normalizedUsername,
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Update user preferences (theme, language, notifications, newsletter)
 */
export const updatePreferences = async (req: Request, res: Response, next: NextFunction) => {
  try {
    if (!req.user) {
      throw new BadRequestError("User not authenticated");
    }

    const userId = req.user.id;
    const { language, themePreference, notifications, newsletterEnabled } = req.body;

    const updateData: any = {};
    const preferenceUpdateData: any = {};

    // Validate and update language
    if (language !== undefined) {
      const validLanguages = ["en", "ar", "es", "fr", "de"];
      if (!validLanguages.includes(language)) {
        throw new BadRequestError(`Invalid language. Must be one of: ${validLanguages.join(", ")}`);
      }
      preferenceUpdateData.language = language;
    }

    // Validate and update theme
    if (themePreference !== undefined) {
      const validThemes = ["light", "dark", "system"];
      if (!validThemes.includes(themePreference)) {
        throw new BadRequestError(`Invalid theme. Must be one of: ${validThemes.join(", ")}`);
      }
      preferenceUpdateData.themePreference = themePreference;
    }

    // Update notifications preference
    if (notifications !== undefined) {
      if (typeof notifications !== "boolean") {
        throw new BadRequestError("Notifications must be a boolean value");
      }
      preferenceUpdateData.notifications = notifications;
    }

    // Update newsletter preference (stored in User table)
    if (newsletterEnabled !== undefined) {
      if (typeof newsletterEnabled !== "boolean") {
        throw new BadRequestError("Newsletter enabled must be a boolean value");
      }
      updateData.newsletterEnabled = newsletterEnabled;
    }

    // Update user preferences and newsletter setting in a transaction
    const result = await prisma.$transaction(async (tx) => {
      // Update or create user preferences if there are preference changes
      let preferences = null;
      if (Object.keys(preferenceUpdateData).length > 0) {
        preferences = await tx.userPreference.upsert({
          where: { userId },
          create: {
            userId,
            language: preferenceUpdateData.language || "en",
            themePreference: preferenceUpdateData.themePreference || "light",
            notifications: preferenceUpdateData.notifications !== undefined ? preferenceUpdateData.notifications : true,
          },
          update: preferenceUpdateData,
        });
      }

      // Update user newsletter setting if provided
      let user = null;
      if (Object.keys(updateData).length > 0) {
        user = await tx.user.update({
          where: { id: userId },
          data: updateData,
          select: {
            id: true,
            newsletterEnabled: true,
          },
        });
      }

      // Fetch complete preferences
      if (!preferences) {
        preferences = await tx.userPreference.findUnique({
          where: { userId },
        });
      }

      return { preferences, user };
    });

    res.status(200).json({
      message: "Preferences updated successfully",
      preferences: result.preferences,
      newsletterEnabled: result.user?.newsletterEnabled,
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Get user preferences
 */
export const getPreferences = async (req: Request, res: Response, next: NextFunction) => {
  try {
    if (!req.user) {
      throw new BadRequestError("User not authenticated");
    }

    const userId = req.user.id;

    const [preferences, user] = await Promise.all([
      prisma.userPreference.findUnique({
        where: { userId },
      }),
      prisma.user.findUnique({
        where: { id: userId },
        select: { newsletterEnabled: true },
      }),
    ]);

    res.status(200).json({
      preferences: preferences || {
        language: "en",
        themePreference: "light",
        notifications: true,
      },
      newsletterEnabled: user?.newsletterEnabled || false,
    });
  } catch (err) {
    next(err);
  }
};
