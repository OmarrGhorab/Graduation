-- AlterEnum
ALTER TYPE "VerificationType" ADD VALUE 'DEVICE_VERIFICATION';

-- AlterTable
ALTER TABLE "User" ADD COLUMN     "deviceBlocked" BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN     "pendingDeviceFingerprint" TEXT;

-- AlterTable
ALTER TABLE "UserDevice" ADD COLUMN     "loginCount" INTEGER NOT NULL DEFAULT 0;
