import { Request, Response, NextFunction } from "express";
import prisma from "../libs/prisma";
import { BadRequestError, UnauthorizedError, NotFoundError } from "../utils/errors";
import {
  generateSecret,
  generateQRCode,
  verifyToken,
  encryptSecret,
  decryptSecret,
  generateBackupCodes,
  encryptBackupCodes,
  verifyBackupCode,
} from "../utils/twoFactor";
import { signAccessToken, signAndStoreRefreshToken } from "../utils/tokens";
import { createSession, getSessionDeviceInfo } from "../utils/sessions";
import bcrypt from "bcrypt";
import dotenv from "dotenv";
dotenv.config();

/**
 * Generate 2FA secret and QR code for enabling 2FA
 */
export const enable2FA = async (req: Request, res: Response, next: NextFunction) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const userId = req.user.id;

    // Get user
    const user = await prisma.user.findUnique({
      where: { id: userId },
      select: { id: true, email: true, twoFactorEnabled: true, twoFactorSecret: true },
    });

    if (!user) {
      throw new NotFoundError("User not found");
    }

    // Check if 2FA is already enabled
    if (user.twoFactorEnabled) {
      throw new BadRequestError("2FA is already enabled");
    }

    // Generate secret
    const serviceName = process.env.SERVICE_NAME || "Auth Service";
    const { secret, otpauthUrl } = generateSecret(user.email, serviceName);

    // Generate QR code
    const qrCodeDataUrl = await generateQRCode(otpauthUrl);

    // Store the secret temporarily (encrypted) - user needs to verify before enabling
    const encryptedSecret = encryptSecret(secret);
    await prisma.user.update({
      where: { id: userId },
      data: { twoFactorSecret: encryptedSecret },
    });

    res.status(200).json({
      message: "2FA secret generated. Please verify with a code from your authenticator app.",
      qrCode: qrCodeDataUrl,
      secret: secret, // Include secret for manual entry (optional)
      manualEntryKey: secret, // For users who can't scan QR code
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Verify 2FA setup and enable it
 */
export const verify2FASetup = async (req: Request, res: Response, next: NextFunction) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const userId = req.user.id;
    const { token } = req.body as { token?: string };

    if (!token) {
      throw new BadRequestError("Token is required");
    }

    // Get user
    const user = await prisma.user.findUnique({
      where: { id: userId },
      select: {
        id: true,
        email: true,
        twoFactorEnabled: true,
        twoFactorSecret: true,
      },
    });

    if (!user) {
      throw new NotFoundError("User not found");
    }

    if (!user.twoFactorSecret) {
      throw new BadRequestError("No 2FA secret found. Please generate a secret first.");
    }

    if (user.twoFactorEnabled) {
      throw new BadRequestError("2FA is already enabled");
    }

    // Decrypt secret
    const secret = decryptSecret(user.twoFactorSecret);

    // Verify token
    const isValid = verifyToken(secret, token);

    if (!isValid) {
      throw new BadRequestError("Invalid token. Please try again.");
    }

    // Generate backup codes
    const backupCodes = generateBackupCodes(10);
    const encryptedBackupCodes = encryptBackupCodes(backupCodes);

    // Enable 2FA
    await prisma.user.update({
      where: { id: userId },
      data: {
        twoFactorEnabled: true,
        twoFactorBackupCodes: encryptedBackupCodes,
      },
    });

    res.status(200).json({
      message: "2FA enabled successfully",
      backupCodes: backupCodes, // Show backup codes only once
      warning: "Please save these backup codes in a safe place. You won't be able to see them again.",
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Disable 2FA
 */
export const disable2FA = async (req: Request, res: Response, next: NextFunction) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const userId = req.user.id;
    const { password, token } = req.body as { password?: string; token?: string };

    // Get user with password
    const user = await prisma.user.findUnique({
      where: { id: userId },
      select: {
        id: true,
        password: true,
        twoFactorEnabled: true,
        twoFactorSecret: true,
      },
    });

    if (!user) {
      throw new NotFoundError("User not found");
    }

    if (!user.twoFactorEnabled) {
      throw new BadRequestError("2FA is not enabled");
    }

    // Verify password
    if (user.password) {
      if (!password) {
        throw new BadRequestError("Password is required to disable 2FA");
      }

      const isValidPassword = await bcrypt.compare(password, user.password);

      if (!isValidPassword) {
        throw new UnauthorizedError("Invalid password");
      }
    }

    // If 2FA is enabled, also verify the token or backup code
    if (user.twoFactorSecret && token) {
      const secret = decryptSecret(user.twoFactorSecret);
      const isValidToken = verifyToken(secret, token);

      if (!isValidToken) {
        // Try backup code
        const userWithBackupCodes = await prisma.user.findUnique({
          where: { id: userId },
          select: { twoFactorBackupCodes: true },
        });

        if (userWithBackupCodes?.twoFactorBackupCodes) {
          const backupResult = verifyBackupCode(userWithBackupCodes.twoFactorBackupCodes, token);
          if (!backupResult.valid) {
            throw new UnauthorizedError("Invalid token or backup code");
          }

          // Update backup codes if a backup code was used
          await prisma.user.update({
            where: { id: userId },
            data: { twoFactorBackupCodes: backupResult.remainingCodes },
          });
        } else {
          throw new UnauthorizedError("Invalid token");
        }
      }
    }

    // Disable 2FA
    await prisma.user.update({
      where: { id: userId },
      data: {
        twoFactorEnabled: false,
        twoFactorSecret: null,
        twoFactorBackupCodes: [],
      },
    });

    res.status(200).json({
      message: "2FA disabled successfully",
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Verify 2FA token during login
 */
export const verify2FALogin = async (req: Request, res: Response, next: NextFunction) => {
  try {
    // Require authentication - user should be authenticated from login step
    if (!req.user) {
      throw new UnauthorizedError("Authentication required. Please login first.");
    }

    const { token, backupCode } = req.body as {
      token?: string;
      backupCode?: string;
    };

    if (!token && !backupCode) {
      throw new BadRequestError("Token or backup code is required");
    }

    // Get user using authenticated user ID
    const user = await prisma.user.findUnique({
      where: { id: req.user.id },
      select: {
        id: true,
        email: true,
        twoFactorEnabled: true,
        twoFactorSecret: true,
        twoFactorBackupCodes: true,
      },
    });

    if (!user) {
      throw new NotFoundError("User not found");
    }

    if (!user.twoFactorEnabled) {
      throw new BadRequestError("2FA is not enabled for this account");
    }

    if (!user.twoFactorSecret) {
      throw new BadRequestError("2FA secret not found");
    }

    // Verify token or backup code
    let isValid = false;
    let remainingBackupCodes = user.twoFactorBackupCodes;

    if (token) {
      const secret = decryptSecret(user.twoFactorSecret);
      isValid = verifyToken(secret, token);
    } else if (backupCode && user.twoFactorBackupCodes) {
      const backupResult = verifyBackupCode(user.twoFactorBackupCodes, backupCode);
      isValid = backupResult.valid;
      remainingBackupCodes = backupResult.remainingCodes;
    }

    if (!isValid) {
      throw new UnauthorizedError("Invalid token or backup code");
    }

    // Update backup codes if a backup code was used
    if (backupCode && remainingBackupCodes !== user.twoFactorBackupCodes) {
      await prisma.user.update({
        where: { id: user.id },
        data: { twoFactorBackupCodes: remainingBackupCodes },
      });
    }

    // Get full user data for login
    const fullUser = await prisma.user.findUnique({
      where: { id: user.id },
      select: {
        id: true,
        name: true,
        username: true,
        email: true,
        verified: true,
        onboardingCompleted: true,
        role: true,
        profileImg: true,
      },
    });

    if (!fullUser) {
      throw new NotFoundError("User not found");
    }

    // Issue tokens and complete login
    const { token: accessToken, jti: accessJti } = signAccessToken({ id: fullUser.id, role: fullUser.role });
    const { token: refreshToken, jti: refreshJti } = await signAndStoreRefreshToken(fullUser.id);
    
    // Create session record in database
    const sessionDeviceInfo = await getSessionDeviceInfo(req);
    const expiresAt = new Date();
    expiresAt.setSeconds(expiresAt.getSeconds() + parseInt(process.env.ACCESS_TOKEN_TTL_SEC || "900", 10));
    const refreshExpiresAt = new Date();
    refreshExpiresAt.setSeconds(refreshExpiresAt.getSeconds() + parseInt(process.env.REFRESH_TOKEN_TTL_SEC || "2592000", 10));
    
    // Find existing session from temp token (created during login) and update it, or create new one
    const existingSession = await prisma.session.findFirst({
      where: {
        userId: fullUser.id,
        sessionToken: req.user?.jti, // Temp token JTI from middleware
      },
    });
    
    if (existingSession) {
      // Update existing temporary session with real tokens
      await prisma.session.update({
        where: { id: existingSession.id },
        data: {
          sessionToken: accessJti,
          refreshToken: refreshJti,
          expiresAt: expiresAt,
          refreshExpiresAt: refreshExpiresAt,
          lastActivityAt: new Date(),
        },
      });
    } else {
      // Create new session if temp session doesn't exist
      // Find device ID from request (if available)
      let deviceId: string | null = null;
      // Try to find device by user agent/IP if needed
      const device = await prisma.userDevice.findFirst({
        where: {
          userId: fullUser.id,
        },
        orderBy: {
          lastLoginAt: "desc",
        },
      });
      if (device) {
        deviceId = device.id;
      }
      
      await createSession({
        userId: fullUser.id,
        deviceId: deviceId,
        sessionToken: accessJti,
        refreshTokenJti: refreshJti,
        ipAddress: sessionDeviceInfo.ipAddress,
        userAgent: sessionDeviceInfo.userAgent,
        location: sessionDeviceInfo.location,
        expiresAt: expiresAt,
        refreshExpiresAt: refreshExpiresAt,
      });
    }

    // Update last login
    await prisma.user.update({
      where: { id: fullUser.id },
      data: { lastLoginAt: new Date() },
    });

    res.status(200).json({
      message: "2FA verification successful",
      user: {
        id: fullUser.id,
        name: fullUser.name,
        username: fullUser.username,
        email: fullUser.email,
        verified: fullUser.verified,
        onboardingCompleted: fullUser.onboardingCompleted,
        role: fullUser.role,
        profileImg: fullUser.profileImg,
      },
      requiresOnboarding: !fullUser.onboardingCompleted,
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Get 2FA status
 */
export const get2FAStatus = async (req: Request, res: Response, next: NextFunction) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const userId = req.user.id;

    const user = await prisma.user.findUnique({
      where: { id: userId },
      select: {
        id: true,
        twoFactorEnabled: true,
        twoFactorBackupCodes: true,
      },
    });

    if (!user) {
      throw new NotFoundError("User not found");
    }

    res.status(200).json({
      twoFactorEnabled: user.twoFactorEnabled,
      backupCodesCount: user.twoFactorBackupCodes?.length || 0,
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Regenerate backup codes
 */
export const regenerateBackupCodes = async (req: Request, res: Response, next: NextFunction) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const userId = req.user.id;
    const { password } = req.body as { password?: string };

    // Get user
    const user = await prisma.user.findUnique({
      where: { id: userId },
      select: {
        id: true,
        password: true,
        twoFactorEnabled: true,
      },
    });

    if (!user) {
      throw new NotFoundError("User not found");
    }

    if (!user.twoFactorEnabled) {
      throw new BadRequestError("2FA is not enabled");
    }

    // Verify password
    if (user.password) {
      if (!password) {
        throw new BadRequestError("Password is required to regenerate backup codes");
      }

      const bcrypt = await import("bcrypt");
      const isValidPassword = await bcrypt.default.compare(password, user.password);

      if (!isValidPassword) {
        throw new UnauthorizedError("Invalid password");
      }
    }

    // Generate new backup codes
    const backupCodes = generateBackupCodes(10);
    const encryptedBackupCodes = encryptBackupCodes(backupCodes);

    // Update backup codes
    await prisma.user.update({
      where: { id: userId },
      data: { twoFactorBackupCodes: encryptedBackupCodes },
    });

    res.status(200).json({
      message: "Backup codes regenerated successfully",
      backupCodes: backupCodes,
      warning: "Please save these backup codes in a safe place. You won't be able to see them again.",
    });
  } catch (err) {
    next(err);
  }
};

