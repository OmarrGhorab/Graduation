import express, { Request, Response } from "express";
import cors from "cors";
import dotenv from "dotenv";
import notificationsRouter from "./routes/notifications.route";
import { errorHandler } from "./middleware/errorHandler";

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

// Error handler last
app.use(errorHandler);

const PORT = process.env.PORT || 6003;

app.listen(PORT, () => {
    console.log(`notification service is running on port ${PORT}`);
});
