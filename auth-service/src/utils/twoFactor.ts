import speakeasy from "speakeasy";
import QRCode from "qrcode";
import crypto from "crypto";
import dotenv from "dotenv";
dotenv.config();

// Encryption key for 2FA secrets (should be in environment variables)
// Generate a key if not provided (for development only - should be set in production)
const getEncryptionKey = (): string => {
  const key = process.env.TWO_FACTOR_ENCRYPTION_KEY;
  if (!key) {
    console.warn("WARNING: TWO_FACTOR_ENCRYPTION_KEY not set. Using a randomly generated key (not persistent across restarts).");
    return crypto.randomBytes(32).toString("hex");
  }
  // Key should be 64 hex characters (32 bytes)
  if (key.length !== 64) {
    throw new Error("TWO_FACTOR_ENCRYPTION_KEY must be 64 hex characters (32 bytes)");
  }
  return key;
};

const ENCRYPTION_KEY = getEncryptionKey();
const ALGORITHM = "aes-256-cbc";
const IV_LENGTH = 16;

/**
 * Encrypt a string using AES-256-CBC
 */
function encrypt(text: string): string {
  const iv = crypto.randomBytes(IV_LENGTH);
  const cipher = crypto.createCipheriv(
    ALGORITHM,
    Buffer.from(ENCRYPTION_KEY, "hex"),
    iv
  );
  let encrypted = cipher.update(text, "utf8", "hex");
  encrypted += cipher.final("hex");
  return iv.toString("hex") + ":" + encrypted;
}

/**
 * Decrypt a string using AES-256-CBC
 */
function decrypt(encryptedText: string): string {
  const parts = encryptedText.split(":");
  const iv = Buffer.from(parts[0], "hex");
  const encrypted = parts[1];
  const decipher = crypto.createDecipheriv(
    ALGORITHM,
    Buffer.from(ENCRYPTION_KEY, "hex"),
    iv
  );
  let decrypted = decipher.update(encrypted, "hex", "utf8");
  decrypted += decipher.final("utf8");
  return decrypted;
}

/**
 * Generate a TOTP secret for a user
 */
export function generateSecret(userEmail: string, serviceName: string = "Auth Service"): {
  secret: string;
  otpauthUrl: string;
} {
  const secret = speakeasy.generateSecret({
    name: `${serviceName} (${userEmail})`,
    issuer: serviceName,
    length: 32,
  });

  return {
    secret: secret.base32 || "",
    otpauthUrl: secret.otpauth_url || "",
  };
}

/**
 * Generate QR code data URL for Google Authenticator
 */
export async function generateQRCode(otpauthUrl: string): Promise<string> {
  try {
    const qrCodeDataUrl = await QRCode.toDataURL(otpauthUrl);
    return qrCodeDataUrl;
  } catch (error) {
    throw new Error("Failed to generate QR code");
  }
}

/**
 * Verify a TOTP token
 */
export function verifyToken(secret: string, token: string, window: number = 2): boolean {
  return speakeasy.totp.verify({
    secret,
    encoding: "base32",
    token,
    window, // Allow 2 time steps (60 seconds) of tolerance
  });
}

/**
 * Encrypt and store a 2FA secret
 */
export function encryptSecret(secret: string): string {
  return encrypt(secret);
}

/**
 * Decrypt a stored 2FA secret
 */
export function decryptSecret(encryptedSecret: string): string {
  return decrypt(encryptedSecret);
}

/**
 * Generate backup codes for 2FA
 */
export function generateBackupCodes(count: number = 10): string[] {
  const codes: string[] = [];
  for (let i = 0; i < count; i++) {
    // Generate 8-character alphanumeric codes
    const code = crypto.randomBytes(4).toString("hex").toUpperCase();
    codes.push(code);
  }
  return codes;
}

/**
 * Encrypt backup codes
 */
export function encryptBackupCodes(codes: string[]): string[] {
  return codes.map((code) => encrypt(code));
}

/**
 * Decrypt backup codes
 */
export function decryptBackupCodes(encryptedCodes: string[]): string[] {
  return encryptedCodes.map((encryptedCode) => decrypt(encryptedCode));
}

/**
 * Verify a backup code
 */
export function verifyBackupCode(
  encryptedCodes: string[],
  code: string
): { valid: boolean; remainingCodes: string[] } {
  const decryptedCodes = decryptBackupCodes(encryptedCodes);
  const index = decryptedCodes.findIndex((c) => c === code.toUpperCase());

  if (index === -1) {
    return { valid: false, remainingCodes: encryptedCodes };
  }

  // Remove used code
  const remainingCodes = [...encryptedCodes];
  remainingCodes.splice(index, 1);

  // Re-encrypt remaining codes
  const decryptedRemaining = decryptBackupCodes(remainingCodes);
  const reEncryptedRemaining = encryptBackupCodes(decryptedRemaining);

  return { valid: true, remainingCodes: reEncryptedRemaining };
}

