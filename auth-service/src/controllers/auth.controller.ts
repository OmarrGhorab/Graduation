import { Request, Response, NextFunction } from "express";
import bcrypt from "bcrypt";
import prisma from "../libs/prisma";
import { BadRequestError, UnauthorizedError } from "../utils/errors";
import { createAndStoreOtp, verifyOtp } from "../utils/otp";
import { signAccessToken, signAndStoreRefreshToken, verifyRefreshToken, revokeRefreshTokenByJti } from "../utils/tokens";
import { setAuthCookies, clearAuthCookies } from "../utils/cookies";
import { aj } from "../libs/arcjet";
import { generateUsernameSuggestions, generateUniqueUsername } from "../utils/username";
import { OAuth2Client } from "google-auth-library";
import dotenv from "dotenv";


dotenv.config();

export const registerUser = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const { name, username, email, password: rawPassword } = req.body as { name?: string; username?: string; email?: string; password?: unknown };
        if (!name || !username || !email || rawPassword === undefined || rawPassword === null) throw new BadRequestError("Missing required fields");
        const decision = await aj.protect(req, { email });
        console.log("Arcjet decision:", decision);

        if (decision.isDenied()) {
            if (decision.reason.isEmail()) {
            return res.status(400).json({ error: "Invalid email" });
            } else {
            return res.status(403).json({ error: "Forbidden" });
            }
        }
        const password = typeof rawPassword === "string" ? rawPassword : String(rawPassword);
        if (password.length < 8) throw new BadRequestError("Password must be at least 8 characters");

        // Check email first
        const existingEmail = await prisma.user.findUnique({ where: { email } });
        if (existingEmail) throw new BadRequestError("Email already in use");

        // Check username separately to provide suggestions if taken
        const existingUsername = await prisma.user.findUnique({ where: { username } });
        if (existingUsername) {
            const suggestions = await generateUsernameSuggestions(username, 3);
            return res.status(400).json({
                error: "Username already taken",
                suggestions: suggestions.length > 0 ? suggestions : undefined,
            });
        }

        const hashed = await bcrypt.hash(password, 10);
        const user = await prisma.user.create({
            data: { name, username, email, password: hashed, verified: false },
        });

        // issue tokens; store refresh in Redis only (no DB storage)
        const accessToken = signAccessToken({ id: user.id, role: user.role });
        const { token: refreshToken } = await signAndStoreRefreshToken(user.id);
        setAuthCookies(res, accessToken, refreshToken);

        // email verification OTP
        const otp = await createAndStoreOtp(`email:${email}`);
        // TODO: send OTP via email provider

        res.status(201).json({
            user: { id: user.id, name: user.name, email: user.email, verified: user.verified },
            message: "Registration successful. Please Verify your account with the OTP Via Email.",
            // expose OTP in non-production for testing
            otp: process.env.NODE_ENV === "production" ? undefined : otp,
        });
    } catch (err) {
        next(err);
    }
}

export const loginUser = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const { emailOrUsername, password: rawPassword } = req.body as { emailOrUsername?: string; password?: unknown };
        if (!emailOrUsername || rawPassword === undefined || rawPassword === null) throw new BadRequestError("Missing credentials");
        const password = typeof rawPassword === "string" ? rawPassword : String(rawPassword);

        const user = await prisma.user.findFirst({ where: { OR: [{ email: emailOrUsername }, { username: emailOrUsername }] } });
        if (!user || !user.password) throw new UnauthorizedError("Invalid credentials");

        const ok = await bcrypt.compare(password, user.password);
        if (!ok) throw new UnauthorizedError("Invalid credentials");

        await prisma.user.update({ where: { id: user.id }, data: { lastLoginAt: new Date() } });

        const accessToken = signAccessToken({ id: user.id, role: user.role });
        const { token: refreshToken } = await signAndStoreRefreshToken(user.id);
        setAuthCookies(res, accessToken, refreshToken);

        res.json({ user: { id: user.id, name: user.name, username: user.username, email: user.email, verified: user.verified } });
    } catch (err) {
        next(err);
    }
}

export const logoutUser = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const refreshToken = (req.headers["x-refresh-token"] as string | undefined);
        if (refreshToken) {
            try {
                const payload = await verifyRefreshToken(refreshToken);
                await revokeRefreshTokenByJti(payload.jti);
            } catch {
                // ignore
            }
        }
        clearAuthCookies(res);
        res.json({ message: "Logged out" });
    } catch (err) {
        next(err);
    }
}

export const forgotPassword = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const { email } = req.body as { email?: string };
        if (!email) throw new BadRequestError("Email is required");

        const user = await prisma.user.findUnique({ where: { email } });
        if (!user) {
            // do not reveal existence
            return res.json({ message: "If the email exists, an OTP has been sent." });
        }
        const otp = await createAndStoreOtp(`reset:${email}`);
        // TODO: send OTP via email provider
        res.json({ message: "If the email exists, an OTP has been sent.", otp: process.env.NODE_ENV === "production" ? undefined : otp });
    } catch (err) {
        next(err);
    }
}

export const resetPassword = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const { email, otp, newPassword } = req.body as { email?: string; otp?: string; newPassword?: string };
        if (!email || !otp || !newPassword) throw new BadRequestError("Missing required fields");

        const ok = await verifyOtp(`reset:${email}`, otp);
        if (!ok) throw new UnauthorizedError("Invalid or expired OTP");

        const hashed = await bcrypt.hash(newPassword, 10);
        await prisma.user.update({ where: { email }, data: { password: hashed } });
        res.json({ message: "Password reset successful" });
    } catch (err) {
        next(err);
    }
}

export const verifyEmailOtp = async (req: Request, res: Response, next: NextFunction) => {
    try {
        const { email, otp } = req.body as { email?: string; otp?: string };
        if (!email || !otp) throw new BadRequestError("Email and OTP are required");

        const ok = await verifyOtp(`email:${email}`, otp);
        if (!ok) throw new UnauthorizedError("Invalid or expired OTP");

        const user = await prisma.user.update({ where: { email }, data: { verified: true } });
        res.json({ message: "Email verified", user: { id: user.id, email: user.email, verified: user.verified } });
    } catch (err) {
        next(err);
    }
}

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

            // Update last login
            await prisma.user.update({
                where: { id: user.id },
                data: { lastLoginAt: new Date(), verified: true },
            });
        }

        // Issue tokens
        const accessToken = signAccessToken({ id: user.id, role: user.role });
        const { token: refreshToken } = await signAndStoreRefreshToken(user.id);
        setAuthCookies(res, accessToken, refreshToken);

        // Redirect to frontend with success (you can customize this redirect URL)
        const frontendUrl = process.env.FRONTEND_URL || "http://localhost:3000";
        res.redirect(`${frontendUrl}/auth/google/callback?success=true&token=${accessToken}`);
    } catch (err) {
        next(err);
    }
}
