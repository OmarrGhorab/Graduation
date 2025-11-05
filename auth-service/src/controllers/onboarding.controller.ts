import { Request, Response, NextFunction } from "express";
import prisma from "../libs/prisma";
import { BadRequestError } from "../utils/errors";
import { Gender, Prisma, UserRole } from "@prisma/client";
import { uploadProfileImage } from "../utils/cloudinaryUpload";

interface OnboardingRequestBody {
  dateOfBirth?: string; // ISO date string
  gender?: Gender;
  country?: string;
  role?: UserRole;
  profileImg?: string; // Base64 string or data URL
  preferences?: {
    language?: string;
    themePreference?: string;
    notifications?: boolean;
  };
  interests?: string[]; // Array of interest names
}

interface UserUpdateData {
  dateOfBirth?: Date;
  gender?: Gender;
  country?: string;
  role?: UserRole;
  profileImg?: string;
  onboardingCompleted: boolean;
}

/**
 * Validates and converts date string to Date object
 */
function validateAndParseDate(dateString: string): Date {
  const date = new Date(dateString);
  if (isNaN(date.getTime())) {
    throw new BadRequestError("Invalid dateOfBirth format. Use ISO date string.");
  }
  return date;
}

/**
 * Validates gender enum value
 */
function validateGender(gender: Gender): void {
  const validGenders: Gender[] = ["MALE", "FEMALE", "OTHER", "PREFER_NOT_TO_SAY"];
  if (!validGenders.includes(gender)) {
    throw new BadRequestError(`Invalid gender. Must be one of: ${validGenders.join(", ")}`);
  }
}

/**
 * Validates user role enum value
 */
function validateRole(role: UserRole): void {
  const validRoles: UserRole[] = ["STUDENT", "TEACHER", "PARENT", "INSTRUCTOR", "ASSISTANT", "HR", "RECRUITER"];
  if (!validRoles.includes(role)) {
    throw new BadRequestError(`Invalid role. Must be one of: ${validRoles.join(", ")}`);
  }
}

/**
 * Builds user update data object from request body
 */
async function buildUserUpdateData(
  body: OnboardingRequestBody,
  userId: string
): Promise<UserUpdateData> {
  const userUpdateData: UserUpdateData = {
    onboardingCompleted: true,
  };

  if (body.dateOfBirth) {
    userUpdateData.dateOfBirth = validateAndParseDate(body.dateOfBirth);
  }

  if (body.gender) {
    validateGender(body.gender);
    userUpdateData.gender = body.gender;
  }

  if (body.role) {
    validateRole(body.role);
    userUpdateData.role = body.role;
  }

  if (body.country) {
    userUpdateData.country = body.country;
  }

  if (body.profileImg) {
    userUpdateData.profileImg = await uploadProfileImage(body.profileImg, userId);
  }

  return userUpdateData;
}

/**
 * Updates or creates user preferences
 */
async function upsertUserPreferences(
  tx: Omit<Prisma.TransactionClient, "$connect" | "$disconnect" | "$on" | "$transaction" | "$use" | "$extends">,
  userId: string,
  preferences: NonNullable<OnboardingRequestBody["preferences"]>
): Promise<void> {
  await tx.userPreference.upsert({
    where: { userId },
    create: {
      userId,
      language: preferences.language || "en",
      themePreference: preferences.themePreference || "light",
      notifications: preferences.notifications !== undefined ? preferences.notifications : true,
    },
    update: {
      language: preferences.language !== undefined ? preferences.language : undefined,
      themePreference: preferences.themePreference !== undefined ? preferences.themePreference : undefined,
      notifications: preferences.notifications !== undefined ? preferences.notifications : undefined,
    },
  });
}

/**
 * Processes and links user interests
 */
async function processUserInterests(
  tx: Omit<Prisma.TransactionClient, "$connect" | "$disconnect" | "$on" | "$transaction" | "$use" | "$extends">,
  userId: string,
  interests: string[]
): Promise<void> {
  // Remove existing interests for this user
  await tx.userInterest.deleteMany({
    where: { userId },
  });

  // Process each interest
  for (const interestName of interests) {
    if (!interestName || typeof interestName !== "string") {
      continue; // Skip invalid entries
    }

    // Find or create the interest
    const interest = await tx.interest.upsert({
      where: { name: interestName.trim() },
      create: { name: interestName.trim() },
      update: {},
    });

    // Link user to interest
    await tx.userInterest.upsert({
      where: {
        userId_interestId: {
          userId,
          interestId: interest.id,
        },
      },
      create: {
        userId,
        interestId: interest.id,
      },
      update: {},
    });
  }
}

export const createOnboarding = async (req: Request, res: Response, next: NextFunction) => {
  try {
    // User is attached by authenticate middleware
    if (!req.user) {
      throw new BadRequestError("User not authenticated");
    }

    const userId = req.user.id;
    const body = req.body as OnboardingRequestBody;

    // Check if user has already completed onboarding
    const existingUser = await prisma.user.findUnique({
      where: { id: userId },
      select: { onboardingCompleted: true },
    });

    if (existingUser?.onboardingCompleted) {
      return res.status(400).json({ error: "Onboarding already completed" });
    }

    // Build user update data (includes profile image upload)
    const userUpdateData = await buildUserUpdateData(body, userId);

    // Use transaction to ensure all updates succeed or fail together
    const result = await prisma.$transaction(async (tx) => {
      // Update user basic info and mark onboarding as completed
      await tx.user.update({
        where: { id: userId },
        data: userUpdateData,
      });

      // Create or update user preferences
      if (body.preferences) {
        await upsertUserPreferences(tx, userId, body.preferences);
      }

      // Handle interests
      if (body.interests && body.interests.length > 0) {
        await processUserInterests(tx, userId, body.interests);
      }

      // Fetch the complete user with relations
      return await tx.user.findUnique({
        where: { id: userId },
        include: {
          preferences: true,
          interests: {
            include: {
              interest: true,
            },
          },
        },
      });
    });

    res.status(200).json({
      message: "Onboarding completed successfully",
      user: {
        id: result?.id,
        name: result?.name,
        username: result?.username,
        email: result?.email,
        dateOfBirth: result?.dateOfBirth,
        gender: result?.gender,
        country: result?.country,
        role: result?.role,
        profileImg: result?.profileImg,
        onboardingCompleted: result?.onboardingCompleted,
        preferences: result?.preferences,
        interests: result?.interests.map((ui) => ({
          id: ui.interest.id,
          name: ui.interest.name,
        })),
      },
    });
  } catch (err) {
    next(err);
  }
};