-- CreateTable
CREATE TABLE "UnlinkRequest" (
    "id" TEXT NOT NULL,
    "childId" TEXT NOT NULL,
    "parentId" TEXT NOT NULL,
    "status" "RequestStatus" NOT NULL DEFAULT 'PENDING',
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    "respondedAt" TIMESTAMP(3),

    CONSTRAINT "UnlinkRequest_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "UnlinkRequest_childId_idx" ON "UnlinkRequest"("childId");

-- CreateIndex
CREATE INDEX "UnlinkRequest_parentId_idx" ON "UnlinkRequest"("parentId");

-- CreateIndex
CREATE INDEX "UnlinkRequest_status_idx" ON "UnlinkRequest"("status");

-- CreateIndex
CREATE UNIQUE INDEX "UnlinkRequest_childId_parentId_key" ON "UnlinkRequest"("childId", "parentId");

-- AddForeignKey
ALTER TABLE "UnlinkRequest" ADD CONSTRAINT "UnlinkRequest_childId_fkey" FOREIGN KEY ("childId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "UnlinkRequest" ADD CONSTRAINT "UnlinkRequest_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;
