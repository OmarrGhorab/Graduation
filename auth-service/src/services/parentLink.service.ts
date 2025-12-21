import prisma from "../libs/prisma";
import { UserRole, RequestStatus } from "@prisma/client";
import { NotFoundError, UnauthorizedError } from "../utils/errors";

/**
 * Get user role with error handling
 */
export async function getUserRole(userId: string) {
  const user = await prisma.user.findUnique({
    where: { id: userId },
    select: { role: true },
  });

  if (!user) {
    throw new NotFoundError("User not found");
  }

  return user.role;
}

/**
 * Get pending link requests for a user
 */
export async function fetchPendingLinkRequests(userId: string, userType: UserRole) {
  if (userType === UserRole.PARENT) {
    // Get requests received by parent
    const requests = await prisma.parentLinkRequest.findMany({
      where: {
        parentId: userId,
        status: RequestStatus.PENDING,
      },
      include: {
        child: {
          select: {
            id: true,
            username: true,
            name: true,
            email: true,
            profileImg: true,
          },
        },
      },
      orderBy: {
        createdAt: "desc",
      },
    });

    return requests.map((req) => ({
      id: req.id,
      status: req.status,
      createdAt: req.createdAt,
      child: req.child,
    }));
  } else {
    // Get requests sent by child
    const requests = await prisma.parentLinkRequest.findMany({
      where: {
        childId: userId,
        status: RequestStatus.PENDING,
      },
      include: {
        parent: {
          select: {
            id: true,
            username: true,
            name: true,
            email: true,
            profileImg: true,
          },
        },
      },
      orderBy: {
        createdAt: "desc",
      },
    });

    return requests.map((req) => ({
      id: req.id,
      status: req.status,
      createdAt: req.createdAt,
      parent: req.parent,
    }));
  }
}

/**
 * Get pending unlink requests for a user
 */
export async function fetchPendingUnlinkRequests(userId: string, userType: UserRole) {
  if (userType === UserRole.PARENT) {
    // Get unlink requests received by parent
    const requests = await prisma.unlinkRequest.findMany({
      where: {
        parentId: userId,
        status: RequestStatus.PENDING,
      },
      include: {
        child: {
          select: {
            id: true,
            username: true,
            name: true,
            email: true,
            profileImg: true,
          },
        },
      },
      orderBy: {
        createdAt: "desc",
      },
    });

    return requests.map((req) => ({
      id: req.id,
      status: req.status,
      createdAt: req.createdAt,
      child: req.child,
    }));
  } else {
    // Get unlink requests sent by child
    const requests = await prisma.unlinkRequest.findMany({
      where: {
        childId: userId,
        status: RequestStatus.PENDING,
      },
      include: {
        parent: {
          select: {
            id: true,
            username: true,
            name: true,
            email: true,
            profileImg: true,
          },
        },
      },
      orderBy: {
        createdAt: "desc",
      },
    });

    return requests.map((req) => ({
      id: req.id,
      status: req.status,
      createdAt: req.createdAt,
      parent: req.parent,
    }));
  }
}

/**
 * Get linked accounts for a user (parents for child, children for parent)
 */
export async function fetchLinkedAccounts(userId: string, userType: UserRole) {
  if (userType === UserRole.PARENT) {
    // Get children linked to this parent
    const links = await prisma.parentChildLink.findMany({
      where: { parentId: userId },
      include: {
        child: {
          select: {
            id: true,
            username: true,
            name: true,
            email: true,
            profileImg: true,
          },
        },
      },
      orderBy: {
        createdAt: "desc",
      },
    });

    return links.map((link) => ({
      id: link.id,
      child: link.child,
      linkedAt: link.createdAt,
    }));
  } else {
    // Get parents linked to this child
    const links = await prisma.parentChildLink.findMany({
      where: { childId: userId },
      include: {
        parent: {
          select: {
            id: true,
            username: true,
            name: true,
            email: true,
            profileImg: true,
          },
        },
      },
      orderBy: {
        createdAt: "desc",
      },
    });

    return links.map((link) => ({
      id: link.id,
      parent: link.parent,
      linkedAt: link.createdAt,
    }));
  }
}

/**
 * Validate user is a parent and return parent info
 */
export async function validateParentUser(parentId: string) {
  const parent = await prisma.user.findUnique({
    where: { id: parentId },
    select: { id: true, role: true, username: true, name: true, profileImg: true },
  });

  if (!parent || parent.role !== UserRole.PARENT) {
    throw new UnauthorizedError("Only parents can perform this action");
  }

  return parent;
}
