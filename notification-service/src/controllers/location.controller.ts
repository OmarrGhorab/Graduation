import { Response, NextFunction } from "express";
import { AuthenticatedRequest } from "../middleware/auth";
import { UnauthorizedError, BadRequestError, ForbiddenError } from "../utils/errors";
import { sendSilentPushNotification } from "../utils/notifications";
import prisma from "../libs/prisma";

const AUTH_SERVICE_URL = process.env.AUTH_SERVICE_URL || "http://localhost:6001";
const INTERNAL_SERVICE_SECRET = process.env.INTERNAL_SERVICE_SECRET || "";

/**
 * Verify parent-child link by calling auth-service
 */
async function verifyParentChildLink(parentId: string, childId: string): Promise<boolean> {
  try {
    const response = await fetch(
      `${AUTH_SERVICE_URL}/api/v1/parent-link/verify-link?parentId=${parentId}&childId=${childId}`,
      {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          "x-internal-service-secret": INTERNAL_SERVICE_SECRET,
        },
      }
    );

    if (response.ok) {
      const data = await response.json();
      return data.linked === true;
    }
    return false;
  } catch (error) {
    console.error("[Location] Error verifying parent-child link:", error);
    return false;
  }
}

/**
 * Request child's location via silent push notification
 * Only parents linked to the child can request location
 */
export const requestChildLocation = async (
  req: AuthenticatedRequest,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const parentId = req.user.id;
    const { childId } = req.params;

    if (!childId) {
      throw new BadRequestError("Child ID is required");
    }

    // Verify parent role
    if (req.user.role !== "PARENT") {
      throw new ForbiddenError("Only parents can request child location");
    }

    // Verify parent-child link exists via auth-service
    const isLinked = await verifyParentChildLink(parentId, childId);
    if (!isLinked) {
      throw new ForbiddenError("You are not linked to this child");
    }

    // Get child's FCM token from FcmToken table (most recent one)
    const fcmToken = await prisma.fcmToken.findFirst({
      where: {
        userId: childId,
      },
      orderBy: { updatedAt: "desc" },
      select: {
        token: true,
        platform: true,
      },
    });

    if (!fcmToken?.token) {
      throw new BadRequestError(
        "Child device not available for location request. The child may not have the app installed or notifications enabled."
      );
    }

    // Send silent push notification to child's device
    await sendSilentPushNotification(fcmToken.token, fcmToken.platform, {
      type: "location_request",
      parentId,
      requestedAt: new Date().toISOString(),
    });

    res.status(200).json({
      success: true,
      message: "Location request sent to child's device",
    });
  } catch (err) {
    next(err);
  }
};
