import express, { Request, Response } from "express";
import cors from "cors";
import dotenv from "dotenv";
import notificationsRouter from "./routes/notifications.route";
import locationRouter from "./routes/location.route";
import { errorHandler } from "./middleware/errorHandler";
import prisma from "./libs/prisma";

dotenv.config();

const app = express();

app.use(express.json());
app.use(express.urlencoded({ extended: true }));

const allowedOrigins = process.env.ALLOWED_ORIGINS 
  ? process.env.ALLOWED_ORIGINS.split(',').map(origin => origin.trim())
  : ["http://localhost:3000"];

app.use(cors({
    origin: allowedOrigins,
    credentials: true,
    allowedHeaders: ["Content-Type", "Authorization"],
}));

app.get("/", async (req: Request, res: Response) => {
    res.send(`notification service is running`);
});

app.use("/api/v1/notifications", notificationsRouter);
app.use("/api/v1/location", locationRouter);

// Error handler last
app.use(errorHandler);

const PORT = process.env.PORT || 6003;

const server = app.listen(PORT, () => {
    console.log(`notification service is running on port ${PORT}`);
});

// Graceful shutdown
const gracefulShutdown = async (signal: string) => {
    console.log(`\n${signal} received. Starting graceful shutdown...`);
    
    // Stop accepting new connections
    server.close(async () => {
        console.log('HTTP server closed');
        
        // Disconnect Prisma
        await prisma.$disconnect();
        console.log('Database connection closed');
        
        process.exit(0);
    });
    
    // Force shutdown after 10 seconds
    setTimeout(() => {
        console.error('Forced shutdown after timeout');
        process.exit(1);
    }, 10000);
};

process.on('SIGTERM', () => gracefulShutdown('SIGTERM'));
process.on('SIGINT', () => gracefulShutdown('SIGINT'));

