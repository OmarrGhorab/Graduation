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
    const parts = imageData.split(",");
    base64Data = parts[1] || imageData;

    // Check if the content is actually a file URI instead of base64
    if (base64Data.startsWith("file://") || base64Data.startsWith("content://")) {
      throw new BadRequestError(
        "Invalid image data: Received a file URI instead of Base64 content. " +
        "Ensure your mobile client is sending the Base64-encoded image."
      );
    }
  }
  // Fix base64 padding if missing
  const paddingLength = (4 - base64Data.length % 4) % 4;
  if (paddingLength > 0) {
    base64Data += "=".repeat(paddingLength);
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

    // Debug logging
    console.log("Image data length:", imageData.length);
    console.log("Base64 data length:", base64Data.length);
    console.log("MIME type:", mimeType);
    console.log("Base64 data preview:", base64Data.substring(0, 100) + "...");
    console.log("Base64 data end:", "..." + base64Data.substring(base64Data.length - 50));
    console.log("Base64 length % 4:", base64Data.length % 4);

    // Validate base64 data
    if (base64Data.length === 0) {
      throw new BadRequestError("Empty base64 data after parsing");
    }

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

    // Construct the data URI for Cloudinary
    const dataUri = `data:${mimeType};base64,${base64Data}`;
    console.log("Data URI length:", dataUri.length);
    console.log("Data URI preview:", dataUri.substring(0, 100) + "...");

    // Upload to Cloudinary
    const result = await cloudinary.uploader.upload(dataUri, uploadOptions);

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
      const nameWithoutExtension = filename.split(".")[0];

      // Reconstruct folder path if needed
      const uploadIndex = urlParts.findIndex((part) => part === "upload");
      if (uploadIndex !== -1) {
        // Path after 'upload' but before filename
        // Usually contains version string (v...) and folders
        const foldersAndVersion = urlParts.slice(uploadIndex + 1, urlParts.length - 1);

        // Remove version string (starts with v and followed by numbers)
        const folders = foldersAndVersion.filter(part => !/^v\d+$/.test(part));

        if (folders.length > 0) {
          extractedPublicId = `${folders.join("/")}/${nameWithoutExtension}`;
        } else {
          extractedPublicId = nameWithoutExtension;
        }
      }
    }

    const result = await cloudinary.uploader.destroy(extractedPublicId);
    return result.result === "ok";
  } catch (error) {
    console.error("Cloudinary delete error:", error);
    return false;
  }
}

