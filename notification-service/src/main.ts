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

// Debug endpoint to check env (REMOVE IN PRODUCTION)
app.get("/debug/env", async (req: Request, res: Response) => {
    res.json({
        hasSecret: !!process.env.INTERNAL_SERVICE_SECRET,
        secretLength: process.env.INTERNAL_SERVICE_SECRET?.length,
        secretPreview: process.env.INTERNAL_SERVICE_SECRET?.substring(0, 20) + "...",
    });
});

// Health check with dependency verification
app.get("/health", async (req: Request, res: Response) => {
    try {
        // Check database connectivity
        const dbHealthy = await prisma.$queryRaw`SELECT 1`.then(() => true).catch(() => false);

        const isHealthy = dbHealthy;

        res.status(isHealthy ? 200 : 503).json({
            status: isHealthy ? "ok" : "degraded",
            service: "notification-service",
            dependencies: {
                database: dbHealthy ? "ok" : "error",
            },
            timestamp: new Date().toISOString(),
        });
    } catch (error) {
        res.status(503).json({
            status: "error",
            service: "notification-service",
            error: "Health check failed",
            timestamp: new Date().toISOString(),
        });
    }
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

    // Set a timeout to force exit if graceful shutdown takes too long
    const forceShutdownTimeout = setTimeout(() => {
        console.error('Forced shutdown after timeout');
        process.exit(1);
    }, 10000);

    try {
        // Disconnect Prisma first or in parallel
        await prisma.$disconnect();
        console.log('Database connection closed');

        // Stop accepting new connections
        // We use close() but we don't necessarily wait for all connections to finish 
        // because SSE connections will keep it open forever.
        server.close((err) => {
            if (err) {
                console.error('Error closing HTTP server:', err);
            } else {
                console.log('HTTP server closed');
            }
            clearTimeout(forceShutdownTimeout);
            process.exit(0);
        });

        // For Node 18.2.0+ we can force close connections
        if ('closeAllConnections' in server) {
            (server as any).closeAllConnections();
        }

    } catch (error) {
        console.error('Error during graceful shutdown:', error);
        process.exit(1);
    }
};

process.on('SIGTERM', () => gracefulShutdown('SIGTERM'));
process.on('SIGINT', () => gracefulShutdown('SIGINT'));

