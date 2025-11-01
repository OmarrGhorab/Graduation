import { Request, Response, NextFunction } from "express";
import bcrypt from "bcrypt";
import prisma from "../libs/prisma";
import { BadRequestError, UnauthorizedError } from "../utils/errors";
import { createAndStoreOtp, verifyOtp } from "../utils/otp";
import { signAccessToken, signAndStoreRefreshToken, verifyRefreshToken, revokeRefreshTokenByJti } from "../utils/tokens";
import { setAuthCookies, clearAuthCookies } from "../utils/cookies";
import { aj } from "../libs/arcjet";


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

        const existing = await prisma.user.findFirst({ where: { OR: [{ email }, { username }] } });
        if (existing) throw new BadRequestError("Email or username already in use");

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
            user: { id: user.id, name: user.name, username: user.username, email: user.email, verified: user.verified },
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
