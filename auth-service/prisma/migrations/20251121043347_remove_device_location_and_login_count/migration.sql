/*
  Warnings:

  - You are about to drop the column `location` on the `UserDevice` table. All the data in the column will be lost.
  - You are about to drop the column `loginCount` on the `UserDevice` table. All the data in the column will be lost.

*/
-- AlterTable
ALTER TABLE "UserDevice" DROP COLUMN "location",
DROP COLUMN "loginCount";

-- CreateIndex
CREATE INDEX "UserDevice_deviceFingerprint_idx" ON "UserDevice"("deviceFingerprint");
