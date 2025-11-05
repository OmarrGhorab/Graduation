import cloudinary from "../libs/cloudinary";
import { BadRequestError } from "./errors";

export interface ImageUploadOptions {
  folder?: string;
  publicId?: string;
  width?: number;
  height?: number;
  crop?: "fill" | "fit" | "scale" | "thumb" | "limit";
  gravity?: "face" | "auto" | "center";
  quality?: "auto" | number;
  overwrite?: boolean;
}

/**
 * Parse base64 image data and extract MIME type
 * @param imageData - Base64 string or data URL
 * @returns Object with base64 data and MIME type
 */
function parseImageData(imageData: string): { base64Data: string; mimeType: string } {
  let base64Data = imageData;
  let mimeType = "image/jpeg"; // Default to JPEG

  if (imageData.startsWith("data:")) {
    // Extract MIME type from data URL
    const mimeMatch = imageData.match(/data:([^;]+);/);
    if (mimeMatch && mimeMatch[1]) {
      mimeType = mimeMatch[1];
    }
    // Extract base64 data
    base64Data = imageData.split(",")[1] || imageData;
  }

  return { base64Data, mimeType };
}

/**
 * Build Cloudinary transformation array from options
 */
function buildTransformations(options: ImageUploadOptions): Array<Record<string, unknown>> {
  const transformations: Array<Record<string, unknown>> = [];

  if (options.width || options.height) {
    const transformation: Record<string, unknown> = {};
    if (options.width) transformation.width = options.width;
    if (options.height) transformation.height = options.height;
    if (options.crop) transformation.crop = options.crop;
    if (options.gravity) transformation.gravity = options.gravity;
    transformations.push(transformation);
  }

  if (options.quality) {
    transformations.push({ quality: options.quality });
  }

  return transformations;
}

/**
 * Upload image to Cloudinary
 * @param imageData - Base64 string or data URL (data:image/...;base64,...)
 * @param options - Upload and transformation options
 * @returns Promise<string> - Cloudinary secure URL
 * @throws BadRequestError if upload fails
 */
export async function uploadImageToCloudinary(
  imageData: string,
  options: ImageUploadOptions = {}
): Promise<string> {
  try {
    if (!imageData || typeof imageData !== "string" || imageData.trim().length === 0) {
      throw new BadRequestError("Invalid image data");
    }

    const { base64Data, mimeType } = parseImageData(imageData);

    // Build upload options
    const uploadOptions: Record<string, unknown> = {
      folder: options.folder,
      public_id: options.publicId || `image_${Date.now()}`,
      overwrite: options.overwrite !== undefined ? options.overwrite : false,
      resource_type: "image",
    };

    // Add transformations if provided
    const transformations = buildTransformations(options);
    if (transformations.length > 0) {
      uploadOptions.transformation = transformations;
    }

    // Upload to Cloudinary
    const result = await cloudinary.uploader.upload(`data:${mimeType};base64,${base64Data}`, uploadOptions);

    return result.secure_url;
  } catch (error) {
    if (error instanceof BadRequestError) {
      throw error;
    }
    console.error("Cloudinary upload error:", error);
    throw new BadRequestError("Failed to upload image to Cloudinary");
  }
}

/**
 * Upload profile image to Cloudinary with optimized settings
 * @param imageData - Base64 string or data URL
 * @param userId - User ID for folder organization
 * @returns Promise<string> - Cloudinary secure URL
 */
export async function uploadProfileImage(
  imageData: string,
  userId: string
): Promise<string> {
  return uploadImageToCloudinary(imageData, {
    folder: `users/${userId}/profile`,
    publicId: `profile_${Date.now()}`,
    width: 400,
    height: 400,
    crop: "fill",
    gravity: "face",
    quality: "auto",
    overwrite: true,
  });
}

/**
 * Delete image from Cloudinary
 * @param publicId - Cloudinary public ID or full URL
 * @returns Promise<boolean> - Success status
 */
export async function deleteImageFromCloudinary(publicId: string): Promise<boolean> {
  try {
    // Extract public_id from URL if full URL is provided
    let extractedPublicId = publicId;
    if (publicId.includes("cloudinary.com")) {
      const urlParts = publicId.split("/");
      const filename = urlParts[urlParts.length - 1];
      extractedPublicId = filename.split(".")[0];
      // Reconstruct folder path if needed
      const folderIndex = urlParts.findIndex((part) => part === "upload");
      if (folderIndex > 0 && folderIndex < urlParts.length - 2) {
        const folderPath = urlParts.slice(folderIndex + 1, -1).join("/");
        extractedPublicId = `${folderPath}/${extractedPublicId}`;
      }
    }

    const result = await cloudinary.uploader.destroy(extractedPublicId);
    return result.result === "ok";
  } catch (error) {
    console.error("Cloudinary delete error:", error);
    return false;
  }
}

