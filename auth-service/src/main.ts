import express, { Request, Response } from "express";
import cors from "cors";
import dotenv from "dotenv";
import authRouter from "./routes/auth.route";

dotenv.config();

const app = express();

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

app.listen(6001, () => {
    console.log("auth service is running on port 6001");
    
});