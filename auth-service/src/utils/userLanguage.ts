import prisma from "../libs/prisma";

/**
 * Get user's preferred language from UserPreference
 * Falls back to 'en' if no preference exists
 */
export const getUserLanguage = async (userId: string): Promise<string> => {
  try {
    const user = await prisma.user.findUnique({
      where: { id: userId },
      include: { preferences: true }
    });

    return user?.preferences?.language || 'en';
  } catch (error) {
    console.error('Error fetching user language:', error);
    return 'en';
  }
};

/**
 * Get user's preferred language by email
 * Used for password reset and other flows where we only have email
 */
export const getUserLanguageByEmail = async (email: string): Promise<string> => {
  try {
    const user = await prisma.user.findUnique({
      where: { email },
      include: { preferences: true }
    });

    return user?.preferences?.language || 'en';
  } catch (error) {
    console.error('Error fetching user language by email:', error);
    return 'en';
  }
};
