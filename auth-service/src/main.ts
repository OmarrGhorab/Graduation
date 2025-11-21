import express, { Request, Response } from "express";
import cors from "cors";
import dotenv from "dotenv";
import authRouter from "./routes/auth.route";
import onboardingRouter from "./routes/onboarding.route";
import parentLinkRouter from "./routes/parent-link.route";
import { errorHandler } from "./middleware/errorHandler";

dotenv.config();

const app = express();

app.use(express.json({ limit: '40mb' }));
app.use(express.urlencoded({ extended: true, limit: '40mb' }));
app.use(cors(
    {
        origin: ["http://localhost:3000", "http://localhost:8080", "null"],
        credentials: true,
        allowedHeaders: ["Content-Type", "Authorization"],
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