import { Request, Response, NextFunction } from "express";
import bcrypt from "bcrypt";
import prisma from "../libs/prisma";
import { BadRequestError, UnauthorizedError } from "../utils/errors";
import { revokeAllUserRefreshTokens } from "../utils/tokens";
import { clearAuthCookies } from "../utils/cookies";

/**
 * Deactivate user account
 * Sets isActive to false and revokes all refresh tokens
 */
export const deactivateAccount = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const userId = req.user?.id;
        
        if (!userId) {
            throw new UnauthorizedError("Authentication required");
        }

        // Get user to verify they exist and are not already deactivated
        const user = await prisma.user.findUnique({
            where: { id: userId },
            select: {
                id: true,
                isActive: true,
                deletedAt: true,
            },
        });

        if (!user) {
            throw new UnauthorizedError("User not found");
        }

        if (!user.isActive) {
            return res.status(400).json({
                error: "Account already deactivated",
                message: "Your account is already deactivated.",
            });
        }

        if (user.deletedAt) {
            throw new UnauthorizedError("Account has been deleted");
        }

        // Revoke all refresh tokens
        await revokeAllUserRefreshTokens(userId);

        // Update user account status
        await prisma.user.update({
            where: { id: userId },
            data: { isActive: false },
        });

        // Clear auth cookies
        clearAuthCookies(res);

        res.json({
            message: "Account deactivated successfully",
            deactivated: true,
        });
    } catch (err) {
        next(err);
    }
};

/**
 * Delete user account (soft delete)
 * Requires password confirmation for security
 */
export const deleteAccount = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const userId = req.user?.id;
        const { password } = req.body as { password?: string };

        if (!userId) {
            throw new UnauthorizedError("Authentication required");
        }

        if (!password) {
            throw new BadRequestError("Password confirmation is required");
        }

        // Get user with password to verify
        const user = await prisma.user.findUnique({
            where: { id: userId },
            select: {
                id: true,
                password: true,
                deletedAt: true,
            },
        });

        if (!user) {
            throw new UnauthorizedError("User not found");
        }

        if (user.deletedAt) {
            return res.status(400).json({
                error: "Account already deleted",
                message: "Your account has already been deleted.",
            });
        }

        // Verify password
        if (!user.password) {
            throw new BadRequestError("Password verification not available for this account");
        }

        const passwordValid = await bcrypt.compare(password, user.password);
        if (!passwordValid) {
            throw new UnauthorizedError("Invalid password");
        }

        // Revoke all refresh tokens
        await revokeAllUserRefreshTokens(userId);

        // Soft delete user account
        await prisma.user.update({
            where: { id: userId },
            data: { 
                deletedAt: new Date(),
                isActive: false, // Also deactivate
            },
        });

        // Clear auth cookies
        clearAuthCookies(res);

        res.json({
            message: "Account deleted successfully",
            deleted: true,
        });
    } catch (err) {
        next(err);
    }
};

