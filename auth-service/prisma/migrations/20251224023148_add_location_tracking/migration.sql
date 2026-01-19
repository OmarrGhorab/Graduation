-- AlterTable
ALTER TABLE "Session" ADD COLUMN     "lastLatitude" DOUBLE PRECISION,
ADD COLUMN     "lastLocationAccuracy" DOUBLE PRECISION,
ADD COLUMN     "lastLocationAddress" TEXT,
ADD COLUMN     "lastLocationTimestamp" TIMESTAMP(3),
ADD COLUMN     "lastLongitude" DOUBLE PRECISION;

-- CreateTable
CREATE TABLE "LocationHistory" (
    "id" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "latitude" DOUBLE PRECISION NOT NULL,
    "longitude" DOUBLE PRECISION NOT NULL,
    "accuracy" DOUBLE PRECISION,
    "address" TEXT,
    "timestamp" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "LocationHistory_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "LocationHistory_userId_idx" ON "LocationHistory"("userId");

-- CreateIndex
CREATE INDEX "LocationHistory_userId_timestamp_idx" ON "LocationHistory"("userId", "timestamp");

-- AddForeignKey
ALTER TABLE "LocationHistory" ADD CONSTRAINT "LocationHistory_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;
