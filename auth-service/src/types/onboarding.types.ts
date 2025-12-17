import { Gender, UserRole } from "@prisma/client";

/**
 * Request body for onboarding completion
 */
export interface OnboardingRequestBody {
  dateOfBirth?: string; // ISO date string
  gender?: Gender;
  country?: string;
  role?: UserRole;
  profileImg?: string; // Base64 string or data URL
  bio?: string; // Max 200 characters
  goals?: string[]; // Max 3 goals
  newsletterEnabled?: boolean;
  preferences?: {
    language?: string;
    themePreference?: string;
    notifications?: boolean;
  };
  interests?: string[]; // Array of interest names
  parentIds?: string[]; // Array of parent IDs to link (optional)
}

/**
 * User update data for onboarding
 */
export interface UserUpdateData {
  dateOfBirth?: Date;
  gender?: Gender;
  country?: string;
  role?: UserRole;
  profileImg?: string;
  bio?: string;
  goals?: string[];
  newsletterEnabled?: boolean;
  onboardingCompleted: boolean;
}

/**
 * Onboarding response
 */
export interface OnboardingResponse {
  message: string;
  user: {
    id: string;
    name: string;
    username: string;
    email: string;
    dateOfBirth: Date | null;
    gender: Gender | null;
    country: string | null;
    role: UserRole;
    profileImg: string | null;
    bio: string | null;
    goals: string[];
    newsletterEnabled: boolean;
    onboardingCompleted: boolean;
    preferences: {
      id: string;
      userId: string;
      language: string | null;
      themePreference: string | null;
      notifications: boolean | null;
    } | null;
    interests: Array<{
      id: string;
      name: string;
    }>;
    parentLinkRequests?: Array<{
      id: string;
      parentId: string;
      status: string;
      createdAt: Date;
    }>;
  };
}


