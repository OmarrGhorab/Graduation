import dotenv from "dotenv";
dotenv.config();

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

app.use("/api/v1/auth", authRouter);
app.use("/api/v1/onboarding", onboardingRouter);
app.use("/api/v1/parent-link", parentLinkRouter);
app.use("/api/v1/profile", profileRouter);
app.use("/api/v1/location", locationRouter);
app.use("/api/v1/internal", internalRouter);
// Error handler last
app.use(errorHandler);

app.listen(6001, () => {
    console.log("auth service is running on port 6001");
});