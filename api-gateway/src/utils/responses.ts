import { Response } from "express";
import { ApiResponse, ErrorResponse } from "../types/index.js";

/**
 * Send error response with proper typing
 */
export const sendErrorResponse = (res: Response, statusCode: number, error: string): void => {
  const errorResponse: ErrorResponse = { error };
  res.status(statusCode).json(errorResponse);
};

/**
 * Send success response with proper typing
 */
export const sendSuccessResponse = (res: Response, message: string): void => {
  const successResponse: ApiResponse = { message };
  res.status(200).json(successResponse);
};
