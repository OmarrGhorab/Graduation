import { UserRole } from "@prisma/client";

// RequestStatus enum type - defined as string literal union
// This matches the Prisma enum definition
export type RequestStatus = "PENDING" | "ACCEPTED" | "DECLINED" | "CANCELLED";

// RequestStatus enum values for runtime use
export const RequestStatus = {
  PENDING: "PENDING" as const,
  ACCEPTED: "ACCEPTED" as const,
  DECLINED: "DECLINED" as const,
  CANCELLED: "CANCELLED" as const,
} as const;

/**
 * Query parameters for searching parents
 */
export interface SearchParentsQuery {
  query?: string; // Search by username, email, or name
  page?: string;
  limit?: string;
}

/**
 * Request body for sending parent link request
 */
export interface SendParentLinkRequestBody {
  parentId: string;
}

/**
 * Request body for responding to a parent link request
 */
export interface RespondToRequestRequestBody {
  requestId: string;
  action: "accept" | "decline";
}

/**
 * Parent search result item
 */
export interface ParentSearchResult {
  id: string;
  username: string;
  name: string;
  email: string;
  profileImg: string | null;
}

/**
 * Paginated parent search response
 */
export interface PaginatedParentSearchResponse {
  data: ParentSearchResult[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
    hasNext: boolean;
    hasPrevious: boolean;
  };
}

/**
 * Parent link request response
 */
export interface ParentLinkRequestResponse {
  id: string;
  parentId: string;
  childId: string;
  status: RequestStatus;
  createdAt: Date;
  updatedAt: Date;
  respondedAt: Date | null;
  parent?: {
    id: string;
    username: string;
    name: string;
    email: string;
    profileImg: string | null;
  };
  child?: {
    id: string;
    username: string;
    name: string;
    email: string;
    profileImg: string | null;
  };
}

/**
 * Linked account response
 */
export interface LinkedAccountResponse {
  id: string;
  parent?: {
    id: string;
    username: string;
    name: string;
    email: string;
    profileImg: string | null;
  };
  child?: {
    id: string;
    username: string;
    name: string;
    email: string;
    profileImg: string | null;
  };
  linkedAt: Date;
}

// Re-export UserRole for convenience
export { UserRole };

