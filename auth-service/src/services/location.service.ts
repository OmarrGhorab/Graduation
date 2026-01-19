import prisma from "../libs/prisma";
import { DeviceLocation, LocationUpdate } from "../types/device-location.types";

/**
 * Update session location data
 * Called on login, token refresh, or dedicated location update endpoint
 */
export async function updateSessionLocation(
  sessionToken: string,
  location: DeviceLocation
): Promise<void> {
  if (location.latitude === null || location.longitude === null) {
    return; // Skip if no valid location
  }

  await prisma.session.updateMany({
    where: {
      sessionToken,
      isActive: true,
      isRevoked: false,
    },
    data: {
      lastLatitude: location.latitude,
      lastLongitude: location.longitude,
      lastLocationAccuracy: location.accuracy,
      lastLocationAddress: location.address,
      lastLocationTimestamp: new Date(),
    },
  });
}

/**
 * Record location in history for child tracking
 * Creates a new entry in LocationHistory table
 */
export async function recordLocationHistory(
  userId: string,
  location: DeviceLocation
): Promise<void> {
  if (location.latitude === null || location.longitude === null) {
    return; // Skip if no valid location
  }

  await prisma.locationHistory.create({
    data: {
      userId,
      latitude: location.latitude,
      longitude: location.longitude,
      accuracy: location.accuracy,
      address: location.address,
    },
  });
}

/**
 * Get location history for a user (for parent tracking)
 * @param userId - User ID to get history for
 * @param limit - Max number of records (default 100)
 * @param since - Only get records after this date
 */
export async function getLocationHistory(
  userId: string,
  limit: number = 100,
  since?: Date
): Promise<LocationUpdate[]> {
  const where: any = { userId };
  
  if (since) {
    where.timestamp = { gte: since };
  }

  const history = await prisma.locationHistory.findMany({
    where,
    orderBy: { timestamp: "desc" },
    take: limit,
    select: {
      latitude: true,
      longitude: true,
      accuracy: true,
      address: true,
      timestamp: true,
    },
  });

  return history;
}

/**
 * Get latest location for a user
 */
export async function getLatestLocation(userId: string): Promise<LocationUpdate | null> {
  // First try to get from active session
  const session = await prisma.session.findFirst({
    where: {
      userId,
      isActive: true,
      isRevoked: false,
      lastLatitude: { not: null },
      lastLongitude: { not: null },
    },
    orderBy: { lastLocationTimestamp: "desc" },
    select: {
      lastLatitude: true,
      lastLongitude: true,
      lastLocationAccuracy: true,
      lastLocationAddress: true,
      lastLocationTimestamp: true,
    },
  });

  if (session?.lastLatitude && session?.lastLongitude) {
    return {
      latitude: session.lastLatitude,
      longitude: session.lastLongitude,
      accuracy: session.lastLocationAccuracy,
      address: session.lastLocationAddress,
      timestamp: session.lastLocationTimestamp || new Date(),
    };
  }

  // Fallback to location history
  const history = await prisma.locationHistory.findFirst({
    where: { userId },
    orderBy: { timestamp: "desc" },
    select: {
      latitude: true,
      longitude: true,
      accuracy: true,
      address: true,
      timestamp: true,
    },
  });

  return history;
}

/**
 * Clean up old location history entries
 * @param userId - User ID to clean up
 * @param olderThan - Delete entries older than this date
 */
export async function cleanupLocationHistory(
  userId: string,
  olderThan: Date
): Promise<number> {
  const result = await prisma.locationHistory.deleteMany({
    where: {
      userId,
      timestamp: { lt: olderThan },
    },
  });

  return result.count;
}

/**
 * Get all children's latest locations for a parent
 */
export async function getChildrenLocations(parentId: string): Promise<Array<{
  childId: string;
  childName: string;
  location: LocationUpdate | null;
}>> {
  // Get all linked children
  const links = await prisma.parentChildLink.findMany({
    where: { parentId },
    include: {
      child: {
        select: {
          id: true,
          name: true,
        },
      },
    },
  });

  // Get latest location for each child
  const results = await Promise.all(
    links.map(async (link) => ({
      childId: link.child.id,
      childName: link.child.name,
      location: await getLatestLocation(link.child.id),
    }))
  );

  return results;
}
