import prisma from "../libs/prisma";
import { BadRequestError, NotFoundError } from "../utils/errors";
import { UserRole, Prisma } from "@prisma/client";
import { publishNotification } from "./notifications";
import { RequestStatus } from "../types/parent-link.types";

/**
 * Helper function to send a parent link request
 * Can be used during onboarding or from settings
 */
export async function sendParentLinkRequestHelper(
  childId: string,
  parentId: string,
  skipNotification = false
): Promise<{
  id: string;
  parentId: string;
  childId: string;
  status: RequestStatus;
  createdAt: Date;
}> {
  if (childId === parentId) {
    throw new BadRequestError("Cannot send request to yourself");
  }

  // Verify parent exists and has PARENT role
  const parent = await prisma.user.findUnique({
    where: { id: parentId },
    select: { id: true, role: true, username: true, name: true, profileImg: true },
  });

  if (!parent) {
    throw new NotFoundError("Parent not found");
  }

  if (parent.role !== UserRole.PARENT) {
    throw new BadRequestError("User is not a parent");
  }

  // Check if already linked
  const existingLink = await prisma.parentChildLink.findUnique({
    where: {
      parentId_childId: {
        parentId,
        childId,
      },
    },
  });

  if (existingLink) {
    throw new BadRequestError("Already linked to this parent");
  }

  // Check if child already has 2 linked parents (limit: 2 parents per child)
  const childLinksCount = await prisma.parentChildLink.count({
    where: { childId },
  });

  if (childLinksCount >= 2) {
    throw new BadRequestError("Child can only link up to 2 parents");
  }

  // Check if request already exists
  const existingRequest = await prisma.parentLinkRequest.findUnique({
    where: {
      childId_parentId: {
        childId,
        parentId,
      },
    },
  });

  if (existingRequest) {
    if (existingRequest.status === RequestStatus.PENDING) {
      throw new BadRequestError("Request already sent and pending");
    } else if (existingRequest.status === RequestStatus.ACCEPTED) {
      throw new BadRequestError("Already linked to this parent");
    }
    // If declined or cancelled, allow resending by updating the request
  }

  // Get child info for notification
  const child = await prisma.user.findUnique({
    where: { id: childId },
    select: { id: true, username: true, name: true, profileImg: true },
  });

  if (!child) {
    throw new NotFoundError("Child not found");
  }

  // Create or update request
  const request = await prisma.parentLinkRequest.upsert({
    where: {
      childId_parentId: {
        childId,
        parentId,
      },
    },
    create: {
      childId,
      parentId,
      status: RequestStatus.PENDING,
    },
    update: {
      status: RequestStatus.PENDING,
      createdAt: new Date(),
      respondedAt: null,
    },
  });

  // Publish real-time notification to parent (unless skipped)
  if (!skipNotification) {
    await publishNotification(parentId, {
      type: "parent_link_request",
      requestId: request.id,
      child: {
        id: child.id,
        username: child.username,
        name: child.name,
        profileImg: child.profileImg,
      },
      createdAt: request.createdAt.toISOString(),
    });
  }

  return request;
}

/**
 * Helper function to send multiple parent link requests
 * Used during onboarding when user provides multiple parent IDs
 */
export async function sendMultipleParentLinkRequests(
  childId: string,
  parentIds: string[],
  skipNotification = false
): Promise<Array<{
  id: string;
  parentId: string;
  status: RequestStatus;
  error?: string;
}>> {
  const results = [];

  for (const parentId of parentIds) {
    try {
      const request = await sendParentLinkRequestHelper(childId, parentId, skipNotification);
      results.push({
        id: request.id,
        parentId: request.parentId,
        status: request.status,
      });
    } catch (error) {
      // Continue with other requests even if one fails
      results.push({
        id: "",
        parentId,
        status: RequestStatus.PENDING,
        error: error instanceof Error ? error.message : "Unknown error",
      });
    }
  }

  return results;
}

/**
 * Helper function to send an unlink request
 * Child requests to unlink from a parent
 */
export async function sendUnlinkRequestHelper(
  childId: string,
  parentId: string,
  skipNotification = false
): Promise<{
  id: string;
  parentId: string;
  childId: string;
  status: RequestStatus;
  createdAt: Date;
}> {
  if (childId === parentId) {
    throw new BadRequestError("Cannot send unlink request to yourself");
  }

  // Verify parent exists and has PARENT role
  const parent = await prisma.user.findUnique({
    where: { id: parentId },
    select: { id: true, role: true, username: true, name: true, profileImg: true },
  });

  if (!parent) {
    throw new NotFoundError("Parent not found");
  }

  if (parent.role !== UserRole.PARENT) {
    throw new BadRequestError("User is not a parent");
  }

  // Check if linked
  const existingLink = await prisma.parentChildLink.findUnique({
    where: {
      parentId_childId: {
        parentId,
        childId,
      },
    },
  });

  if (!existingLink) {
    throw new BadRequestError("Not linked to this parent");
  }

  // Check if unlink request already exists
  const existingRequest = await prisma.unlinkRequest.findUnique({
    where: {
      childId_parentId: {
        childId,
        parentId,
      },
    },
  });

  if (existingRequest) {
    if (existingRequest.status === RequestStatus.PENDING) {
      throw new BadRequestError("Unlink request already sent and pending");
    }
    // If declined or cancelled, allow resending by updating the request
  }

  // Get child info for notification
  const child = await prisma.user.findUnique({
    where: { id: childId },
    select: { id: true, username: true, name: true, profileImg: true },
  });

  if (!child) {
    throw new NotFoundError("Child not found");
  }

  // Create or update request
  const request = await prisma.unlinkRequest.upsert({
    where: {
      childId_parentId: {
        childId,
        parentId,
      },
    },
    create: {
      childId,
      parentId,
      status: RequestStatus.PENDING,
    },
    update: {
      status: RequestStatus.PENDING,
      createdAt: new Date(),
      respondedAt: null,
    },
  });

  // Publish real-time notification to parent (unless skipped)
  if (!skipNotification) {
    await publishNotification(parentId, {
      type: "unlink_request",
      requestId: request.id,
      child: {
        id: child.id,
        username: child.username,
        name: child.name,
        profileImg: child.profileImg,
      },
      createdAt: request.createdAt.toISOString(),
    });
  }

  return request;
}

