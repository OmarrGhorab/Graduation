import dotenv from "dotenv";
dotenv.config();

import express, { Request, Response } from "express";
import cors from "cors";
import authRouter from "./routes/auth.route";
import onboardingRouter from "./routes/onboarding.route";
import parentLinkRouter from "./routes/parent-link.route";
import { errorHandler } from "./middleware/errorHandler";

const app = express();

app.use(express.json({ limit: '10mb' }));
app.use(express.urlencoded({ extended: true, limit: '10mb' }));
app.use(cors(
    {
        origin: ["http://localhost:3000", "http://localhost:8080"],
        credentials: false, // Not needed for mobile - tokens sent via Authorization header
        allowedHeaders: ["Content-Type", "Authorization", "x-refresh-token"],
    }
));

app.get("/", async (req: Request, res: Response) => {
    res.send(`auth service is running`);
});

app.use("/api/v1/auth", authRouter);
app.use("/api/v1/onboarding", onboardingRouter);
app.use("/api/v1/parent-link", parentLinkRouter);
// Error handler last
app.use(errorHandler);

app.listen(6001, () => {
    console.log("auth service is running on port 6001");
});