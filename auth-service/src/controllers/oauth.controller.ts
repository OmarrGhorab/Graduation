import { Request, Response, NextFunction } from "express";
import prisma from "../libs/prisma";
import { BadRequestError } from "../utils/errors";
import { signAccessToken, signAndStoreRefreshToken } from "../utils/tokens";
import { setAuthCookies } from "../utils/cookies";
import { generateUniqueUsername } from "../utils/username";
import { extractDeviceInfo, extractDeviceName } from "../utils/device";
import { createSession, getSessionDeviceInfo } from "../utils/sessions";
import { OAuth2Client } from "google-auth-library";
import dotenv from "dotenv";

dotenv.config();

/**
 * Initiates Google OAuth flow - redirects user to Google
 */
export const googleAuth = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const clientId = process.env.GOOGLE_CLIENT_ID;
        const clientSecret = process.env.GOOGLE_CLIENT_SECRET;
        const redirectUri = process.env.GOOGLE_REDIRECT_URI;

        if (!clientId || !clientSecret || !redirectUri) {
            throw new BadRequestError("Google OAuth credentials not configured");
        }

        const oauth2Client = new OAuth2Client(clientId, clientSecret, redirectUri);

        // Generate the authorization URL
        const authUrl = oauth2Client.generateAuthUrl({
            access_type: "offline",
            scope: [
                "https://www.googleapis.com/auth/userinfo.profile",
                "https://www.googleapis.com/auth/userinfo.email",
            ],
            prompt: "consent", // Force consent to get refresh token
        });

        res.redirect(authUrl);
    } catch (err) {
        next(err);
    }
}

/**
 * Handles Google OAuth callback - creates or logs in user
 */
export const googleCallback = async (req: Request, res: Response, next: NextFunction) => {
    try {
        
        // Handle code parameter - try multiple ways to extract it
        let code: string | undefined;
        
        // Method 1: Direct from req.query (Express default)
        const queryCode = req.query.code;
        if (queryCode) {
            if (Array.isArray(queryCode)) {
                code = typeof queryCode[0] === 'string' ? queryCode[0] : undefined;
            } else if (typeof queryCode === 'string') {
                code = queryCode;
            }
        }
        
        // Method 2: If not found, try parsing from URL directly
        if (!code && req.url) {
            const queryString = req.url.split('?')[1] || '';
            const urlParams = new URLSearchParams(queryString);
            code = urlParams.get('code') || undefined;
        }
        
        if (!code || typeof code !== 'string') {
            console.error("Code parameter missing or invalid:", { 
                code, 
                type: typeof code, 
                query: req.query,
                hasCodeInQuery: !!req.query.code,
                url: req.url
            });
            throw new BadRequestError("Authorization code not provided");
        }
        

        const clientId = process.env.GOOGLE_CLIENT_ID;
        const clientSecret = process.env.GOOGLE_CLIENT_SECRET;
        const redirectUri = process.env.GOOGLE_REDIRECT_URI;

        if (!clientId || !clientSecret || !redirectUri) {
            throw new BadRequestError("Google OAuth credentials not configured");
        }

        const oauth2Client = new OAuth2Client(clientId, clientSecret, redirectUri);

        // Exchange code for tokens
        const { tokens } = await oauth2Client.getToken(code);
        oauth2Client.setCredentials(tokens);

        // Get user info from Google
        const ticket = await oauth2Client.verifyIdToken({
            idToken: tokens.id_token!,
            audience: clientId,
        });

        const payload = ticket.getPayload();
        if (!payload) throw new BadRequestError("Failed to get user info from Google");

        const googleId = payload.sub;
        const email = payload.email;
        const name = payload.name || "";
        const picture = payload.picture;

        if (!email) throw new BadRequestError("Email not provided by Google");

        // Check if user exists by email
        let user = await prisma.user.findUnique({
            where: { email },
            include: { providers: true },
        });

        // If user doesn't exist, create new user
        if (!user) {
            // Generate unique username from name or email
            const baseUsername = name || email.split("@")[0];
            const username = await generateUniqueUsername(baseUsername);

            user = await prisma.user.create({
                data: {
                    name,
                    username,
                    email,
                    profileImg: picture || undefined,
                    verified: true, // Google email is already verified
                    providers: {
                        create: {
                            provider: "GOOGLE",
                            providerId: googleId,
                            accessToken: tokens.access_token || undefined,
                            refreshToken: tokens.refresh_token || undefined,
                        },
                    },
                },
                include: { providers: true },
            });
        } else {
            // User exists - check if Google provider is linked
            const googleProvider = user.providers.find((p) => p.provider === "GOOGLE");

            if (googleProvider) {
                // Update existing provider tokens
                await prisma.authProvider.update({
                    where: { id: googleProvider.id },
                    data: {
                        accessToken: tokens.access_token || undefined,
                        refreshToken: tokens.refresh_token || undefined,
                    },
                });
            } else {
                // Link Google provider to existing user
                await prisma.authProvider.create({
                    data: {
                        provider: "GOOGLE",
                        providerId: googleId,
                        userId: user.id,
                        accessToken: tokens.access_token || undefined,
                        refreshToken: tokens.refresh_token || undefined,
                    },
                });
            }

            // Update user profile image if available
            if (picture && !user.profileImg) {
                await prisma.user.update({
                    where: { id: user.id },
                    data: { profileImg: picture },
                });
            }

            // Update verified status (lastLoginAt will be updated after session creation)
            await prisma.user.update({
                where: { id: user.id },
                data: { verified: true },
            });
            // Keep local user state in sync for subsequent checks
            user.verified = true;
        }

        // Check if user account is verified (should always be true for Google users, but check for safety)
        if (!user.verified) {
            const frontendUrl = process.env.FRONTEND_URL || "http://localhost:3000";
            return res.redirect(`${frontendUrl}/auth/google/callback?error=verification_required&message=Please verify your account`);
        }

        // Issue tokens
        const { token: accessToken, jti: accessJti } = signAccessToken({ id: user.id, role: user.role });
        const { token: refreshToken, jti: refreshJti } = await signAndStoreRefreshToken(user.id);
        
        // Create session record in database
        const sessionDeviceInfo = await getSessionDeviceInfo(req);
        const expiresAt = new Date();
        expiresAt.setSeconds(expiresAt.getSeconds() + parseInt(process.env.ACCESS_TOKEN_TTL_SEC || "900", 10));
        const refreshExpiresAt = new Date();
        refreshExpiresAt.setSeconds(refreshExpiresAt.getSeconds() + parseInt(process.env.REFRESH_TOKEN_TTL_SEC || "2592000", 10));
        
        // Extract device info and find or create device
        const deviceInfo = extractDeviceInfo(req);
        let deviceId: string | null = null;
        
        // Try to find existing device
        const existingDevice = await prisma.userDevice.findUnique({
            where: {
                userId_deviceFingerprint: {
                    userId: user.id,
                    deviceFingerprint: deviceInfo.fingerprint,
                },
            },
        });
        
        if (existingDevice) {
            deviceId = existingDevice.id;
            // Update device login count and last login
            await prisma.userDevice.update({
                where: { id: existingDevice.id },
                data: {
                    lastLoginAt: new Date(),
                },
            });
        } else {
            // Create new device
            const deviceName = extractDeviceName(deviceInfo.userAgent);
            const newDevice = await prisma.userDevice.create({
                data: {
                    userId: user.id,
                    deviceFingerprint: deviceInfo.fingerprint,
                    deviceName: deviceName,
                    platform: deviceInfo.platform,
                    ipAddress: deviceInfo.ipAddress,
                    isTrusted: true, // Google OAuth users are trusted by default
                    lastLoginAt: new Date(),
                },
            });
            deviceId = newDevice.id;
        }
        
        // Create session
        await createSession({
            userId: user.id,
            deviceId: deviceId,
            sessionToken: accessJti,
            refreshTokenJti: refreshJti,
            ipAddress: sessionDeviceInfo.ipAddress,
            userAgent: sessionDeviceInfo.userAgent,
            location: sessionDeviceInfo.location,
            expiresAt: expiresAt,
            refreshExpiresAt: refreshExpiresAt,
        });
        
        setAuthCookies(res, accessToken, refreshToken);
        
        // Update last login
        await prisma.user.update({
            where: { id: user.id },
            data: { lastLoginAt: new Date() },
        });

        // Check if onboarding is completed
        if (!user.onboardingCompleted) {
            const frontendUrl = process.env.FRONTEND_URL || "http://localhost:3000";
            return res.redirect(`${frontendUrl}/auth/google/callback?success=true&token=${accessToken}&requiresOnboarding=true`);
        }

        // Redirect to frontend with success (you can customize this redirect URL)
        const frontendUrl = process.env.FRONTEND_URL || "http://localhost:3000";
        res.redirect(`${frontendUrl}/auth/google/callback?success=true&token=${accessToken}`);
    } catch (err) {
        next(err);
    }
}
