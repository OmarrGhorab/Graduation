import path from "path";
import dotenv from "dotenv";
dotenv.config({ path: path.resolve(process.cwd(), ".env") });

const secret = process.env.INTERNAL_SERVICE_SECRET || "";
const jwtSecret = process.env.JWT_ACCESS_SECRET || "";
console.log(`[Auth Service] Internal Secret: ${secret.substring(0, 5)}...${secret.substring(secret.length - 4)}`);
console.log(`[Auth Service] JWT Secret: ${jwtSecret.substring(0, 5)}...${jwtSecret.substring(jwtSecret.length - 4)}`);

import express, { Request, Response } from "express";
import cors from "cors";
import authRouter from "./routes/auth.route";
import onboardingRouter from "./routes/onboarding.route";
import parentLinkRouter from "./routes/parent-link.route";
import profileRouter from "./routes/profile.route";
import locationRouter from "./routes/location.route";
import internalRouter from "./routes/internal.route";
import { errorHandler } from "./middleware/errorHandler";
import { extractDeviceInfo } from "./middleware/deviceInfo.middleware";
import prisma from "./libs/prisma";
import redis from "./libs/redis";

const app = express();

// Trust proxy for correct IP detection behind reverse proxies
app.set("trust proxy", true);

app.use(express.json({ limit: '10mb' }));
app.use(express.urlencoded({ extended: true, limit: '10mb' }));
app.use(cors({
    origin: true, // Allow all origins (mobile apps don't send origin header)
    credentials: false, // Not needed for mobile - tokens sent via Authorization header
    allowedHeaders: [
        "Content-Type",
        "Authorization",
        "x-refresh-token",
        "x-forwarded-for",
        "x-real-ip",
        // Custom device info headers
        "x-device-name",
        "x-device-model",
        "x-device-os-version",
        "x-app-version",
        "x-device-location",
        "x-device-timezone",
        "x-device-platform",
        // GPS location headers
        "x-device-latitude",
        "x-device-longitude",
        "x-device-location-accuracy",
    ],
}));

// Extract device info from headers on all requests
app.use(extractDeviceInfo);

app.get("/", async (req: Request, res: Response) => {
    res.send(`auth service is running`);
});

// OPTIMIZED: Health check with dependency verification
app.get("/health", async (req: Request, res: Response) => {
    try {
        // Check database connectivity
        const dbHealthy = await prisma.$queryRaw`SELECT 1`.then(() => true).catch(() => false);
        
        // Check Redis connectivity
        const redisHealthy = await redis.ping().then(() => true).catch(() => false);
        
        const isHealthy = dbHealthy && redisHealthy;
        
        res.status(isHealthy ? 200 : 503).json({
            status: isHealthy ? "ok" : "degraded",
            service: "auth-service",
            dependencies: {
                database: dbHealthy ? "ok" : "error",
                redis: redisHealthy ? "ok" : "error",
            },
            timestamp: new Date().toISOString(),
        });
    } catch (error) {
        res.status(503).json({
            status: "error",
            service: "auth-service",
            error: "Health check failed",
            timestamp: new Date().toISOString(),
        });
    }
});

app.use("/api/v1/auth", authRouter);
app.use("/api/v1/onboarding", onboardingRouter);
app.use("/api/v1/parent-link", parentLinkRouter);
app.use("/api/v1/profile", profileRouter);
app.use("/api/v1/location", locationRouter);
app.use("/api/v1/internal", internalRouter);
// Error handler last
app.use(errorHandler);

const PORT = process.env.PORT || 6001;

const server = app.listen(PORT, () => {
    console.log(`auth service is running on port ${PORT}`);
});

// OPTIMIZED: Graceful shutdown handling
const gracefulShutdown = async (signal: string) => {
    console.log(`\n${signal} received. Starting graceful shutdown...`);
    
    // Stop accepting new connections
    server.close(async () => {
        console.log("HTTP server closed");
        
        try {
            // Disconnect Prisma
            await prisma.$disconnect();
            console.log("Database connection closed");
            
            // Disconnect Redis
            await redis.quit();
            console.log("Redis connection closed");
            
            process.exit(0);
        } catch (error) {
            console.error("Error during shutdown:", error);
            process.exit(1);
        }
    });
    
    // Force shutdown after 10 seconds
    setTimeout(() => {
        console.error("Forced shutdown after timeout");
        process.exit(1);
    }, 10000);
};

process.on("SIGTERM", () => gracefulShutdown("SIGTERM"));
process.on("SIGINT", () => gracefulShutdown("SIGINT"));