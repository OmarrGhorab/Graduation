/*
  Warnings:

  - You are about to drop the `PasswordReset` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `Verification` table. If the table is not empty, all the data it contains will be lost.

*/
-- DropForeignKey
ALTER TABLE "public"."PasswordReset" DROP CONSTRAINT "PasswordReset_userId_fkey";

-- DropForeignKey
ALTER TABLE "public"."Verification" DROP CONSTRAINT "Verification_userId_fkey";

-- AlterTable
ALTER TABLE "UserPreference" ALTER COLUMN "language" SET DEFAULT 'system',
ALTER COLUMN "themePreference" SET DEFAULT 'system';

-- DropTable
DROP TABLE "public"."PasswordReset";

-- DropTable
DROP TABLE "public"."Verification";

-- DropEnum
DROP TYPE "public"."VerificationType";

-- CreateIndex
CREATE INDEX "LocationHistory_timestamp_idx" ON "LocationHistory"("timestamp");

-- CreateIndex
CREATE INDEX "Session_userId_isActive_isRevoked_expiresAt_idx" ON "Session"("userId", "isActive", "isRevoked", "expiresAt");

-- CreateIndex
CREATE INDEX "Session_expiresAt_idx" ON "Session"("expiresAt");
