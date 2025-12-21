-- AlterTable
ALTER TABLE "User" ADD COLUMN     "bio" VARCHAR(200),
ADD COLUMN     "goals" TEXT[] DEFAULT ARRAY[]::TEXT[],
ADD COLUMN     "newsletterEnabled" BOOLEAN NOT NULL DEFAULT false;
