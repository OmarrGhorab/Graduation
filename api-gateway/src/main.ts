import dotenv from "dotenv";
dotenv.config();

import express, { Request, Response, NextFunction } from "express";
import cors from "cors";
import proxy from "express-http-proxy";
import arcjet, { detectBot } from "@arcjet/node";

// Initialize Arcjet with bot detection
const aj = arcjet({
    key: process.env.ARCJET_KEY || "",
    rules: [
        detectBot({
            mode: process.env.NODE_ENV === "production" ? "LIVE" : "DRY_RUN",
            allow: [
                "CATEGORY:SEARCH_ENGINE", // Google, Bing, etc
                "CATEGORY:MONITOR", // Uptime monitoring services
            ],
        }),
    ],
});

const app = express();

// Middleware
app.use(express.json({ limit: "10mb" }));
app.use(express.urlencoded({ extended: true, limit: "10mb" }));
// CORS configuration - allow mobile apps and web clients
const allowedOrigins = process.env.ALLOWED_ORIGINS
    ? process.env.ALLOWED_ORIGINS.split(',').map(origin => origin.trim())
    : ["http://localhost:3000", "http://localhost:8080", 'http://10.0.2.2'];

app.use(cors({
    origin: (origin, callback) => {
        // Allow requests with no origin (mobile apps, Postman, etc.)
        if (!origin) {
            return callback(null, true);
        }
        // Allow if origin is in whitelist
        if (allowedOrigins.includes(origin) || allowedOrigins.includes("*")) {
            return callback(null, true);
        }
        callback(new Error("Not allowed by CORS"));
    },
    credentials: true,
    allowedHeaders: ["Content-Type", "Authorization", "x-refresh-token"],
}));

// Health check (skip protection)
app.get("/health", (req: Request, res: Response) => {
    res.json({ status: "ok", service: "api-gateway" });
});

// Arcjet protection middleware (bot + VPN/proxy/hosting detection)
const arcjetProtection = async (req: Request, res: Response, next: NextFunction) => {
    // Skip protection if no Arcjet key configured or if not in production
    const isProd = process.env.NODE_ENV === "production";
    if (!process.env.ARCJET_KEY || !isProd) {
        return next();
    }

    try {
        const decision = await aj.protect(req);

        // Block if request is denied (bot detected)
        if (decision.isDenied()) {
            if (decision.reason.isBot()) {
                console.log("Bot detected, blocking request");
                return res.status(403).json({ error: "Forbidden: Bot detected" });
            }
            return res.status(403).json({ error: "Forbidden" });
        }

        // Block VPN, proxy, hosting, and relay IPs
        if (
            decision.ip.isHosting() ||
            decision.ip.isVpn() ||
            decision.ip.isProxy() ||
            decision.ip.isRelay()
        ) {
            console.log("VPN/Proxy/Hosting detected, blocking request");
            return res.status(403).json({ error: "Forbidden: VPN/Proxy not allowed" });
        }

        // Request passed all checks
        next();
    } catch (error) {
        console.error("Arcjet protection error:", error);
        // On error, allow request through (fail open)
        next();
    }
};

// Apply Arcjet protection to all routes except health check
app.use(arcjetProtection);

// Notification service proxy
app.use("/api/v1/notifications", proxy("http://localhost:6003"));

// Auth service proxy (everything else)
app.use("/", proxy("http://localhost:6001"));

const PORT = process.env.PORT || 3000;

app.listen(PORT, () => {
    console.log(`api-gateway is running on port ${PORT}`);
});
