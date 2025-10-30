import express, { Request, Response } from "express";
import cors from "cors";
import dotenv from "dotenv";
import authRouter from "./routes/auth.route";
import { errorHandler } from "./middleware/errorHandler";

dotenv.config();

const app = express();
app.set("trust proxy", 1);

app.use(express.json());
app.use(express.urlencoded({ extended: true }));
app.use(cors(
    {
        origin: "http://localhost:3000",
        credentials: true,
        allowedHeaders: ["Content-Type", "Authorization"],
    }
));

app.get("/", async (req: Request, res: Response) => {
    res.send(`auth service is running`);
});

app.use("/api/v1/auth", authRouter);

// Error handler last
app.use(errorHandler);

app.listen(6001, () => {
    console.log("auth service is running on port 6001");
    
});