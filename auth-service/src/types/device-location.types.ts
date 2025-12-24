/**
 * Device location data extracted from HTTP headers
 * Headers may be empty/missing if user denied location permissions
 */
export interface DeviceLocation {
  latitude: number | null;
  longitude: number | null;
  accuracy: number | null; // Accuracy in meters (< 50m = GPS, > 1000m = network-based)
  address: string | null; // Full reverse-geocoded address
  timezone: string | null;
}

/**
 * Full device context extracted from HTTP headers
 */
export interface DeviceInfo {
  name: string | null;
  model: string | null;
  platform: 'android' | 'ios' | null;
  osVersion: string | null;
  appVersion: string | null;
  ipAddress: string | null;
  userAgent: string | null;
}

/**
 * Combined device and location info for session tracking
 */
export interface DeviceContext {
  location: DeviceLocation;
  device: DeviceInfo;
}

/**
 * Location update payload for session/history
 */
export interface LocationUpdate {
  latitude: number;
  longitude: number;
  accuracy: number | null;
  address: string | null;
  timestamp: Date;
}
