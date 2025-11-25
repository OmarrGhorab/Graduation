import { PrismaClient } from '@prisma/client'
import dotenv from "dotenv";

dotenv.config();

const prisma = new PrismaClient({
  log: process.env.NODE_ENV === 'development' ? ['query', 'info', 'warn', 'error'] : ['error'],
})

// Handle database connection errors
prisma.$connect()
  .then(() => {
    console.log('Connected to database');
  })
  .catch((error: Error) => {
    console.error('Database connection error:', error);
    process.exit(1);
  });

// Graceful shutdown
process.on('beforeExit', async () => {
  await prisma.$disconnect();
});

export default prisma
