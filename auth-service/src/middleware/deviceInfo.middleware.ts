import { Request, Response, NextFunction } from "express";
import { Platform } from "@prisma/client";

export interface DeviceInfo {
  deviceName: string;
  deviceModel: string | null;
  osVersion: string | null;
  appVersion: string | null;
  location: string | null;
  timezone: string;
  platform: Platform;
  ipAddress: string;
  userAgent: string | null;
}

export interface DeviceLocationData {
  latitude: number | null;
  longitude: number | null;
  accuracy: number | null;
  address: string | null;
  timezone: string | null;
}

// Extend Express Request type
declare global {
  namespace Express {
    interface Request {
      deviceInfo?: DeviceInfo;
      deviceLocation?: DeviceLocationData;
    }
  }
}

/**
 * Get client IP address from request headers
 * Priority: X-Forwarded-For > X-Real-IP > req.ip
 * Filters out private IPs when possible
 */
function getClientIp(req: Request): string {
  // Check X-Forwarded-For header (common for proxies/load balancers)
  const forwardedFor = req.headers["x-forwarded-for"];
  if (forwardedFor) {
    const ip = Array.isArray(forwardedFor)
      ? forwardedFor[0]
      : forwardedFor.split(",")[0].trim();
    
    // Return if it's a public IP
    if (!isPrivateIp(ip)) {
      return ip;
    }
  }

  // Check X-Real-IP header
  const realIp = req.headers["x-real-ip"];
  if (realIp) {
    const ip = Array.isArray(realIp) ? realIp[0] : realIp;
    if (!isPrivateIp(ip)) {
      return ip;
    }
  }

  // Fallback to req.ip
  const reqIp = req.ip?.replace("::ffff:", "") || "0.0.0.0";
  return reqIp;
}

/**
 * Check if IP is private/local
 */
function isPrivateIp(ip: string): boolean {
  return (
    ip.startsWith("192.168.") ||
    ip.startsWith("10.") ||
    ip.startsWith("172.16.") ||
    ip.startsWith("172.17.") ||
    ip.startsWith("172.18.") ||
    ip.startsWith("172.19.") ||
    ip.startsWith("172.20.") ||
    ip.startsWith("172.21.") ||
    ip.startsWith("172.22.") ||
    ip.startsWith("172.23.") ||
    ip.startsWith("172.24.") ||
    ip.startsWith("172.25.") ||
    ip.startsWith("172.26.") ||
    ip.startsWith("172.27.") ||
    ip.startsWith("172.28.") ||
    ip.startsWith("172.29.") ||
    ip.startsWith("172.30.") ||
    ip.startsWith("172.31.") ||
    ip === "127.0.0.1" ||
    ip === "::1" ||
    ip === "localhost"
  );
}

/**
 * Get header value as string
 */
function getHeader(req: Request, name: string): string | null {
  const value = req.headers[name];
  if (!value) return null;
  return typeof value === "string" ? value : value[0] || null;
}

/**
 * Parse float from header value, returns null if invalid
 */
function parseFloatHeader(req: Request, name: string): number | null {
  const value = getHeader(req, name);
  if (!value) return null;
  const parsed = parseFloat(value);
  return isNaN(parsed) ? null : parsed;
}

/**
 * Extract GPS location data from request headers
 */
function extractLocationData(req: Request): DeviceLocationData {
  return {
    latitude: parseFloatHeader(req, "x-device-latitude"),
    longitude: parseFloatHeader(req, "x-device-longitude"),
    accuracy: parseFloatHeader(req, "x-device-location-accuracy"),
    address: getHeader(req, "x-device-location"),
    timezone: getHeader(req, "x-device-timezone"),
  };
}

/**
 * Parse platform from header value
 */
function parsePlatform(platformHeader: string | null, userAgent: string | null): Platform {
  if (platformHeader) {
    const normalized = platformHeader.toUpperCase();
    if (normalized === "ANDROID") return Platform.ANDROID;
    if (normalized === "IOS") return Platform.IOS;
    if (normalized === "DESKTOP") return Platform.DESKTOP;
    if (normalized === "WEB") return Platform.WEB;
  }

  // Fallback: detect from user agent
  if (userAgent) {
    const ua = userAgent.toLowerCase();
    if (ua.includes("android")) return Platform.ANDROID;
    if (ua.includes("iphone") || ua.includes("ipad")) return Platform.IOS;
    if (ua.includes("windows") || ua.includes("macintosh") || ua.includes("linux")) {
      return Platform.WEB;
    }
  }

  return Platform.WEB;
}

/**
 * Middleware to extract device information from request headers
 * Attaches deviceInfo and deviceLocation objects to req for use in controllers
 */
export const extractDeviceInfo = (req: Request, res: Response, next: NextFunction) => {
  const userAgent = req.headers["user-agent"] || null;
  const platformHeader = getHeader(req, "x-device-platform");

  req.deviceInfo = {
    deviceName: getHeader(req, "x-device-name") || "Unknown Device",
    deviceModel: getHeader(req, "x-device-model"),
    osVersion: getHeader(req, "x-device-os-version"),
    appVersion: getHeader(req, "x-app-version"),
    location: getHeader(req, "x-device-location"),
    timezone: getHeader(req, "x-device-timezone") || "UTC",
    platform: parsePlatform(platformHeader, userAgent),
    ipAddress: getClientIp(req),
    userAgent,
  };

  // Extract GPS location data
  req.deviceLocation = extractLocationData(req);

  next();
};

/**
 * Helper function to get device info from request (for use outside middleware)
 */
export function getDeviceInfoFromRequest(req: Request): DeviceInfo {
  if (req.deviceInfo) {
    return req.deviceInfo;
  }

  const userAgent = req.headers["user-agent"] || null;
  const platformHeader = getHeader(req, "x-device-platform");

  return {
    deviceName: getHeader(req, "x-device-name") || "Unknown Device",
    deviceModel: getHeader(req, "x-device-model"),
    osVersion: getHeader(req, "x-device-os-version"),
    appVersion: getHeader(req, "x-app-version"),
    location: getHeader(req, "x-device-location"),
    timezone: getHeader(req, "x-device-timezone") || "UTC",
    platform: parsePlatform(platformHeader, userAgent),
    ipAddress: getClientIp(req),
    userAgent,
  };
}

/**
 * Helper function to get device location from request (for use outside middleware)
 */
export function getDeviceLocationFromRequest(req: Request): DeviceLocationData {
  if (req.deviceLocation) {
    return req.deviceLocation;
  }
  return extractLocationData(req);
}

/**
 * Check if location data is valid (has both lat and lng)
 */
export function hasValidLocation(location: DeviceLocationData): boolean {
  return location.latitude !== null && location.longitude !== null;
}
