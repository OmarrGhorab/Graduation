import { Request } from "express";
import crypto from "crypto";
import { Platform } from "@prisma/client";

export interface DeviceInfo {
    fingerprint: string;
    userAgent: string | null;
    ipAddress: string | null;
    platform: Platform | null;
    deviceName?: string | null;
}

/**
 * Get client IP address from request
 */
export function getClientIp(req: Request): string | null {
    // Check various headers for IP (in order of preference)
    const forwarded = req.headers["x-forwarded-for"];
    if (forwarded) {
        if (typeof forwarded === "string") {
            return forwarded.split(",")[0].trim();
        }
        return forwarded[0]?.split(",")[0].trim() || null;
    }

    const realIp = req.headers["x-real-ip"];
    if (realIp) {
        return typeof realIp === "string" ? realIp : realIp[0] || null;
    }

    const remoteAddress = req.socket.remoteAddress;
    if (remoteAddress) {
        // Remove IPv6 prefix if present
        return remoteAddress.replace("::ffff:", "");
    }

    return null;
}

/**
 * Determine platform from user agent string
 */
export function detectPlatform(userAgent: string | null | undefined): Platform | null {
    if (!userAgent) return null;

    const ua = userAgent.toLowerCase();

    // Mobile detection
    if (/android/.test(ua)) {
        return Platform.ANDROID;
    }

    if (/iphone|ipad|ipod/.test(ua)) {
        return Platform.IOS;
    }

    // Desktop detection
    if (/windows|macintosh|linux/.test(ua)) {
        return Platform.DESKTOP;
    }

    // Web/browser detection (fallback)
    if (/mozilla|chrome|safari|firefox|edge/.test(ua)) {
        return Platform.WEB;
    }

    return null;
}

/**
 * Generate device fingerprint from user agent and other device info
 * This creates a consistent identifier for a device
 */
export function generateDeviceFingerprint(
    userAgent: string | null | undefined,
    ipAddress: string | null | undefined,
    acceptLanguage?: string | null,
    acceptEncoding?: string | null
): string {
    // Collect device characteristics
    const components: string[] = [];

    // User agent (normalized - remove version numbers for stability)
    if (userAgent) {
        // Remove version numbers to make fingerprint more stable
        const normalizedUA = userAgent
            .replace(/\d+\.\d+\.\d+/g, "x.x.x")
            .replace(/\d+\.\d+/g, "x.x")
            .toLowerCase();
        components.push(normalizedUA);
    }

    // Platform detection
    const platform = detectPlatform(userAgent || undefined);
    if (platform) {
        components.push(platform);
    }

    // Accept language (first language only)
    if (acceptLanguage) {
        const lang = acceptLanguage.split(",")[0].trim().toLowerCase();
        components.push(lang);
    }

    // Accept encoding (normalized)
    if (acceptEncoding) {
        const encoding = acceptEncoding.split(",")[0].trim().toLowerCase();
        components.push(encoding);
    }

    // Note: We don't include IP address in fingerprint as it can change
    // We'll use IP separately for security checks

    // Create hash from components
    const fingerprintString = components.join("|");
    const hash = crypto.createHash("sha256").update(fingerprintString).digest("hex");

    return hash.substring(0, 32); // Use first 32 chars for shorter fingerprint
}

/**
 * Extract device information from request
 */
export function extractDeviceInfo(req: Request, deviceName?: string): DeviceInfo {
    const userAgent = req.headers["user-agent"] || null;
    const ipAddress = getClientIp(req);
    const acceptLanguage = req.headers["accept-language"] || null;
    const acceptEncoding = req.headers["accept-encoding"] || null;

    const fingerprint = generateDeviceFingerprint(
        userAgent,
        ipAddress,
        acceptLanguage || undefined,
        acceptEncoding || undefined
    );

    const platform = detectPlatform(userAgent || undefined);

    return {
        fingerprint,
        userAgent,
        ipAddress,
        platform,
        deviceName: deviceName || null,
    };
}

/**
 * Extract device name from user agent (for display purposes)
 */
export function extractDeviceName(userAgent: string | null | undefined): string | null {
    if (!userAgent) return null;

    // Try to extract device model for mobile devices
    const androidMatch = userAgent.match(/\(Linux; Android [^)]+; ([^)]+)\)/);
    if (androidMatch) {
        return androidMatch[1];
    }

    const iosMatch = userAgent.match(/\(iPhone; CPU iPhone OS [^)]+ like Mac OS X\)/);
    if (iosMatch) {
        return "iPhone";
    }

    const ipadMatch = userAgent.match(/\(iPad; CPU OS [^)]+ like Mac OS X\)/);
    if (ipadMatch) {
        return "iPad";
    }

    // For desktop, try to extract OS
    const windowsMatch = userAgent.match(/Windows NT ([^)]+)/);
    if (windowsMatch) {
        return `Windows ${windowsMatch[1]}`;
    }

    const macMatch = userAgent.match(/Mac OS X ([^_)]+)/);
    if (macMatch) {
        return `macOS ${macMatch[1]}`;
    }

    const linuxMatch = userAgent.match(/Linux/);
    if (linuxMatch) {
        return "Linux";
    }

    return null;
}


