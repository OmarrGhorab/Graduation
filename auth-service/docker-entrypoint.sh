#!/bin/sh
set -e

npx prisma migrate deploy --schema=./prisma/schema.prisma
npx tsx src/main.ts
