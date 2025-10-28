import express, { Request, Response } from "express";
import dotenv from "dotenv";

dotenv.config();

const app = express();

app.get("/", async (req: Request, res: Response) => {
    res.send(`auth service is running`);
});

app.listen(6001, () => {
    console.log("auth service is running on port 6001");
    
});