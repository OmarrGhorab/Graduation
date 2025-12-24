import { Request } from "express";
import { DeviceLocation, DeviceInfo } from "../types/device-location.types";

/**
 * Get header value as string from request
 */
function getHeader(req: Request, name: string): string | null {
  const value = req.headers[name.toLowerCase()];
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
 * Extract device location from request headers
 * 
 * Expected headers:
 * - X-Device-Latitude: GPS latitude coordinate (e.g., "31.0298633")
 * - X-Device-Longitude: GPS longitude coordinate (e.g., "30.4838118")
 * - X-Device-Location-Accuracy: Accuracy in meters (e.g., "20")
 * - X-Device-Location: Full reverse-geocoded address
 * - X-Device-Timezone: User's timezone (e.g., "Africa/Cairo")
 */
export function extractDeviceLocation(req: Request): DeviceLocation {
  return {
    latitude: parseFloatHeader(req, "x-device-latitude"),
    longitude: parseFloatHeader(req, "x-device-longitude"),
    accuracy: parseFloatHeader(req, "x-device-location-accuracy"),
    address: getHeader(req, "x-device-location"),
    timezone: getHeader(req, "x-device-timezone"),
  };
}

/**
 * Extract full device info from request headers
 * 
 * Expected headers:
 * - X-Device-Name: Device name (e.g., "Redmi 9T")
 * - X-Device-Model: Device model identifier (e.g., "M2010J19SG")
 * - X-Device-Platform: "android" or "ios"
 * - X-Device-OS-Version: OS version (e.g., "Android 12")
 * - X-App-Version: App version (e.g., "1.0.0")
 * - X-Forwarded-For: Public IP address
 * - User-Agent: Custom user agent
 */
export function extractDeviceInfo(req: Request): DeviceInfo {
  const platformHeader = getHeader(req, "x-device-platform")?.toLowerCase();
  let platform: 'android' | 'ios' | null = null;
  
  if (platformHeader === "android") {
    platform = "android";
  } else if (platformHeader === "ios") {
    platform = "ios";
  }

  // Get IP address from various headers
  const forwardedFor = getHeader(req, "x-forwarded-for");
  const ipAddress = forwardedFor 
    ? forwardedFor.split(",")[0].trim() 
    : req.ip?.replace("::ffff:", "") || null;

  return {
    name: getHeader(req, "x-device-name"),
    model: getHeader(req, "x-device-model"),
    platform,
    osVersion: getHeader(req, "x-device-os-version"),
    appVersion: getHeader(req, "x-app-version"),
    ipAddress,
    userAgent: req.headers["user-agent"] || null,
  };
}

/**
 * Check if location data is valid (has both lat and lng)
 */
export function hasValidLocation(location: DeviceLocation): boolean {
  return location.latitude !== null && location.longitude !== null;
}

/**
 * Check if location is precise (GPS-based, accuracy < 50m)
 */
export function isPreciseLocation(location: DeviceLocation): boolean {
  return hasValidLocation(location) && 
         location.accuracy !== null && 
         location.accuracy < 50;
}

/**
 * Check if location is approximate (network-based, accuracy > 1000m)
 */
export function isApproximateLocation(location: DeviceLocation): boolean {
  return hasValidLocation(location) && 
         location.accuracy !== null && 
         location.accuracy > 1000;
}
