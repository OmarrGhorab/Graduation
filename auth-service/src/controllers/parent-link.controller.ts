import { Request, Response, NextFunction } from "express";
import prisma from "../libs/prisma";
import { BadRequestError, NotFoundError, UnauthorizedError } from "../utils/errors";
import { UserRole, Prisma } from "@prisma/client";
import { publishNotification as publishNotificationUtil } from "../utils/notifications";
import { sendParentLinkRequestHelper, sendUnlinkRequestHelper } from "../utils/parent-link";
import {
  RequestStatus,
  type SearchParentsQuery,
  type SendParentLinkRequestBody,
  type RespondToRequestRequestBody,
  type SendUnlinkRequestBody,
  type RespondToUnlinkRequestBody,
} from "../types/parent-link.types";

/**
 * Search for parents by username or email (paginated)
 * Only returns users with PARENT role
 */
export const searchParents = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const { query, page = "1", limit = "10" } = req.query as unknown as SearchParentsQuery;
    const currentUserId = req.user.id;

    const pageNum = parseInt(page, 10);
    const limitNum = parseInt(limit, 10);

    if (pageNum < 1 || limitNum < 1 || limitNum > 50) {
      throw new BadRequestError("Invalid pagination parameters");
    }

    const skip = (pageNum - 1) * limitNum;

    // Build search condition
    const searchCondition = query
      ? {
          OR: [
            { username: { contains: query, mode: "insensitive" as const } },
            { email: { contains: query, mode: "insensitive" as const } },
            { name: { contains: query, mode: "insensitive" as const } },
          ],
        }
      : {};

    // Find parents matching the search, excluding current user
    const [parents, total] = await Promise.all([
      prisma.user.findMany({
        where: {
          ...searchCondition,
          role: UserRole.PARENT,
          id: { not: currentUserId }, // Exclude current user
        },
        select: {
          id: true,
          username: true,
          name: true,
          email: true,
          profileImg: true,
        },
        skip,
        take: limitNum,
        orderBy: {
          createdAt: "desc",
        },
      }),
      prisma.user.count({
        where: {
          ...searchCondition,
          role: UserRole.PARENT,
          id: { not: currentUserId },
        },
      }),
    ]);

    const totalPages = Math.ceil(total / limitNum);

    res.status(200).json({
      data: parents,
      pagination: {
        page: pageNum,
        limit: limitNum,
        total,
        totalPages,
        hasNext: pageNum < totalPages,
        hasPrevious: pageNum > 1,
      },
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Send a parent link request
 */
export const sendParentLinkRequest = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const childId = req.user.id;
    const { parentId } = req.body as SendParentLinkRequestBody;

    if (!parentId) {
      throw new BadRequestError("Parent ID is required");
    }

    // Use helper function to send the request (notifications will be sent)
    const request = await sendParentLinkRequestHelper(childId, parentId, false);

    // Fetch the full request with parent details for response
    const fullRequest = await prisma.parentLinkRequest.findUnique({
      where: { id: request.id },
      include: {
        parent: {
          select: {
            id: true,
            username: true,
            name: true,
            profileImg: true,
          },
        },
      },
    });

    res.status(201).json({
      message: "Parent link request sent successfully",
      request: {
        id: fullRequest!.id,
        parent: fullRequest!.parent,
        status: fullRequest!.status,
        createdAt: fullRequest!.createdAt,
      },
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Get pending requests for the authenticated user (if parent)
 */
export const getPendingRequests = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const userId = req.user.id;

    // Get user role
    const user = await prisma.user.findUnique({
      where: { id: userId },
      select: { role: true },
    });

    if (!user) {
      throw new NotFoundError("User not found");
    }

    if (user.role === UserRole.PARENT) {
      // Get requests received by parent
      const parentRequests = await prisma.parentLinkRequest.findMany({
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

      res.status(200).json({
        data: parentRequests.map((req) => ({
          id: req.id,
          status: req.status,
          createdAt: req.createdAt,
          child: req.child,
        })),
      });
    } else {
      // Get requests sent by child
      const childRequests = await prisma.parentLinkRequest.findMany({
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

      res.status(200).json({
        data: childRequests.map((req) => ({
          id: req.id,
          status: req.status,
          createdAt: req.createdAt,
          parent: req.parent,
        })),
      });
    }
  } catch (err) {
    next(err);
  }
};

/**
 * Accept or decline a parent link request (parent only)
 */
export const respondToRequest = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const parentId = req.user.id;
    const { requestId, action } = req.body as RespondToRequestRequestBody;

    if (!requestId || !action) {
      throw new BadRequestError("Request ID and action are required");
    }

    if (action !== "accept" && action !== "decline") {
      throw new BadRequestError("Action must be 'accept' or 'decline'");
    }

    // Verify user is a parent and get their info
    const parent = await prisma.user.findUnique({
      where: { id: parentId },
      select: { id: true, role: true, username: true, name: true },
    });

    if (!parent || parent.role !== UserRole.PARENT) {
      throw new UnauthorizedError("Only parents can respond to requests");
    }

    // Find the request
    const request = await prisma.parentLinkRequest.findUnique({
      where: { id: requestId },
      include: {
        child: {
          select: {
            id: true,
            username: true,
            name: true,
          },
        },
      },
    });

    if (!request) {
      throw new NotFoundError("Request not found");
    }

    if (request.parentId !== parentId) {
      throw new UnauthorizedError("Not authorized to respond to this request");
    }

    if (request.status !== RequestStatus.PENDING) {
      throw new BadRequestError("Request has already been responded to");
    }

    // Use transaction to update request and create link if accepted
    const result = await prisma.$transaction(async (tx) => {
      const newStatus =
        action === "accept" ? RequestStatus.ACCEPTED : RequestStatus.DECLINED;

      // If accepting, check if parent already has 5 linked children (limit: 5 children per parent)
      if (action === "accept") {
        const parentLinksCount = await tx.parentChildLink.count({
          where: { parentId },
        });

        if (parentLinksCount >= 5) {
          throw new BadRequestError("Parent can only link up to 5 children");
        }
      }

      // Update request status
      const updatedRequest = await tx.parentLinkRequest.update({
        where: { id: requestId },
        data: {
          status: newStatus,
          respondedAt: new Date(),
        },
      });

      // If accepted, create parent-child link
      if (action === "accept") {
        await tx.parentChildLink.create({
          data: {
            parentId: request.parentId,
            childId: request.childId,
          },
        });
      }

      return updatedRequest;
    });

    // Publish real-time notification to child
    await publishNotificationUtil(request.childId, {
      type: `parent_link_request_${action}ed`,
      requestId: request.id,
      parent: {
        id: parent.id,
        username: parent.username,
        name: parent.name,
      },
      status: result.status,
      respondedAt: result.respondedAt?.toISOString(),
    });

    res.status(200).json({
      message: `Request ${action}ed successfully`,
      request: {
        id: result.id,
        status: result.status,
        respondedAt: result.respondedAt,
      },
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Get linked parents (for child) or linked children (for parent)
 */
export const getLinkedAccounts = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const userId = req.user.id;

    const user = await prisma.user.findUnique({
      where: { id: userId },
      select: { role: true },
    });

    if (!user) {
      throw new NotFoundError("User not found");
    }

    let links;

    if (user.role === UserRole.PARENT) {
      // Get children linked to this parent
      links = await prisma.parentChildLink.findMany({
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

      res.status(200).json({
        data: links.map((link) => ({
          id: link.id,
          child: link.child,
          linkedAt: link.createdAt,
        })),
      });
    } else {
      // Get parents linked to this child
      links = await prisma.parentChildLink.findMany({
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

      res.status(200).json({
        data: links.map((link) => ({
          id: link.id,
          parent: link.parent,
          linkedAt: link.createdAt,
        })),
      });
    }
  } catch (err) {
    next(err);
  }
};

/**
 * Send an unlink request (child requests to unlink from a parent)
 */
export const sendUnlinkRequest = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const childId = req.user.id;
    const { parentId } = req.body as SendUnlinkRequestBody;

    if (!parentId) {
      throw new BadRequestError("Parent ID is required");
    }

    // Use helper function to send the unlink request (notifications will be sent)
    const request = await sendUnlinkRequestHelper(childId, parentId, false);

    // Fetch the full request with parent details for response
    const fullRequest = await prisma.unlinkRequest.findUnique({
      where: { id: request.id },
      include: {
        parent: {
          select: {
            id: true,
            username: true,
            name: true,
            profileImg: true,
          },
        },
      },
    });

    res.status(201).json({
      message: "Unlink request sent successfully",
      request: {
        id: fullRequest!.id,
        parent: fullRequest!.parent,
        status: fullRequest!.status,
        createdAt: fullRequest!.createdAt,
      },
    });
  } catch (err) {
    next(err);
  }
};

/**
 * Get pending unlink requests for the authenticated user (if parent)
 */
export const getPendingUnlinkRequests = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const userId = req.user.id;

    // Get user role
    const user = await prisma.user.findUnique({
      where: { id: userId },
      select: { role: true },
    });

    if (!user) {
      throw new NotFoundError("User not found");
    }

    if (user.role === UserRole.PARENT) {
      // Get unlink requests received by parent
      const parentRequests = await prisma.unlinkRequest.findMany({
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

      res.status(200).json({
        data: parentRequests.map((unlinkReq) => ({
          id: unlinkReq.id,
          status: unlinkReq.status,
          createdAt: unlinkReq.createdAt,
          child: unlinkReq.child,
        })),
      });
    } else {
      // Get unlink requests sent by child
      const childRequests = await prisma.unlinkRequest.findMany({
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

      res.status(200).json({
        data: childRequests.map((unlinkReq) => ({
          id: unlinkReq.id,
          status: unlinkReq.status,
          createdAt: unlinkReq.createdAt,
          parent: unlinkReq.parent,
        })),
      });
    }
  } catch (err) {
    next(err);
  }
};

/**
 * Accept or decline an unlink request (parent only)
 */
export const respondToUnlinkRequest = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    if (!req.user) {
      throw new UnauthorizedError("User not authenticated");
    }

    const parentId = req.user.id;
    const { requestId, action } = req.body as RespondToUnlinkRequestBody;

    if (!requestId || !action) {
      throw new BadRequestError("Request ID and action are required");
    }

    if (action !== "accept" && action !== "decline") {
      throw new BadRequestError("Action must be 'accept' or 'decline'");
    }

    // Verify user is a parent and get their info
    const parent = await prisma.user.findUnique({
      where: { id: parentId },
      select: { id: true, role: true, username: true, name: true },
    });

    if (!parent || parent.role !== UserRole.PARENT) {
      throw new UnauthorizedError("Only parents can respond to unlink requests");
    }

    // Find the unlink request
    const request = await prisma.unlinkRequest.findUnique({
      where: { id: requestId },
      include: {
        child: {
          select: {
            id: true,
            username: true,
            name: true,
          },
        },
      },
    });

    if (!request) {
      throw new NotFoundError("Unlink request not found");
    }

    if (request.parentId !== parentId) {
      throw new UnauthorizedError("Not authorized to respond to this unlink request");
    }

    if (request.status !== RequestStatus.PENDING) {
      throw new BadRequestError("Unlink request has already been responded to");
    }

    // Use transaction to update request and delete link if accepted
    const result = await prisma.$transaction(async (tx) => {
      const newStatus =
        action === "accept" ? RequestStatus.ACCEPTED : RequestStatus.DECLINED;

      // Update request status
      const updatedRequest = await tx.unlinkRequest.update({
        where: { id: requestId },
        data: {
          status: newStatus,
          respondedAt: new Date(),
        },
      });

      // If accepted, delete parent-child link
      if (action === "accept") {
        await tx.parentChildLink.delete({
          where: {
            parentId_childId: {
              parentId: request.parentId,
              childId: request.childId,
            },
          },
        });
      }

      return updatedRequest;
    });

    // Publish real-time notification to child
    await publishNotificationUtil(request.childId, {
      type: `unlink_request_${action}ed`,
      requestId: request.id,
      parent: {
        id: parent.id,
        username: parent.username,
        name: parent.name,
      },
      status: result.status,
      respondedAt: result.respondedAt?.toISOString(),
    });

    res.status(200).json({
      message: `Unlink request ${action}ed successfully`,
      request: {
        id: result.id,
        status: result.status,
        respondedAt: result.respondedAt,
      },
    });
  } catch (err) {
    next(err);
  }
};


