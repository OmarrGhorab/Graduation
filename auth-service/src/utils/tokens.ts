import jwt, { type Secret, type SignOptions } from "jsonwebtoken";
import crypto from "crypto";
import redis from "../libs/redis";

type JwtPayload = {
  sub: string; // userId
  jti: string; // token id
  role?: string;
  type: "access" | "refresh";
};

const ACCESS_TOKEN_TTL_SEC = parseInt(process.env.ACCESS_TOKEN_TTL_SEC || "900", 10); // 15 minutes
const REFRESH_TOKEN_TTL_SEC = parseInt(process.env.REFRESH_TOKEN_TTL_SEC || "2592000", 10); // 30 days

const ACCESS_TOKEN_SECRET: Secret = process.env.ACCESS_TOKEN_SECRET || "dev-access-secret";
const REFRESH_TOKEN_SECRET: Secret = process.env.REFRESH_TOKEN_SECRET || "dev-refresh-secret";

export function generateJti(): string {
  return crypto.randomUUID();
}

export function signAccessToken(user: { id: string; role?: string }): { token: string; jti: string } {
  const jti = generateJti();
  const payload: JwtPayload = {
    sub: user.id,
    jti: jti,
    role: user.role,
    type: "access",
  };
  const opts: SignOptions = { expiresIn: ACCESS_TOKEN_TTL_SEC, algorithm: "HS256" };
  const token = jwt.sign(payload, ACCESS_TOKEN_SECRET, opts);
  return { token, jti };
}

export async function signAndStoreRefreshToken(userId: string): Promise<{ token: string; jti: string }> {
  const jti = generateJti();
  const payload: JwtPayload = { sub: userId, jti, type: "refresh" };
  const opts: SignOptions = { expiresIn: REFRESH_TOKEN_TTL_SEC, algorithm: "HS256" };
  const token = jwt.sign(payload, REFRESH_TOKEN_SECRET, opts);

  // Store mapping jti -> userId with TTL
  const key = `rt:${jti}`;
  await redis.set(key, userId, "EX", REFRESH_TOKEN_TTL_SEC);
  
  // Also store jti in user's refresh token set for bulk revocation
  const userTokensKey = `user:${userId}:refresh_tokens`;
  await redis.sadd(userTokensKey, jti);
  await redis.expire(userTokensKey, REFRESH_TOKEN_TTL_SEC);
  
  return { token, jti };
}

export async function verifyAccessToken(token: string): Promise<JwtPayload> {
  const decoded = jwt.verify(token, ACCESS_TOKEN_SECRET) as JwtPayload;
  return decoded;
}

export async function verifyRefreshToken(token: string): Promise<JwtPayload> {
  const decoded = jwt.verify(token, REFRESH_TOKEN_SECRET) as JwtPayload;
  if (decoded.type !== "refresh") throw new Error("Invalid token type");
  const exists = await redis.get(`rt:${decoded.jti}`);
  if (!exists) throw new Error("Refresh token revoked or expired");
  return decoded;
}

export async function revokeRefreshTokenByJti(jti: string): Promise<void> {
  const key = `rt:${jti}`;
  const userId = await redis.get(key);
  await redis.del(key);
  
  // Remove from user's refresh token set if exists
  if (userId) {
    const userTokensKey = `user:${userId}:refresh_tokens`;
    await redis.srem(userTokensKey, jti);
  }
}

export async function rotateRefreshToken(oldJti: string, userId: string): Promise<{ token: string; jti: string }>{
  await revokeRefreshTokenByJti(oldJti);
  return signAndStoreRefreshToken(userId);
}

/**
 * Revoke all refresh tokens for a user
 * This is used when deactivating or deleting an account
 */
export async function revokeAllUserRefreshTokens(userId: string): Promise<void> {
  const userTokensKey = `user:${userId}:refresh_tokens`;
  const jtis = await redis.smembers(userTokensKey);
  
  // Delete all refresh token keys
  if (jtis.length > 0) {
    const keys = jtis.map(jti => `rt:${jti}`);
    await redis.del(...keys);
  }
  
  // Delete the user's refresh token set
  await redis.del(userTokensKey);
}


