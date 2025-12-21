import arcjet, { protectSignup } from "@arcjet/node";

const isProd = process.env.NODE_ENV === "production";

export const aj = arcjet({
  key: process.env.ARCJET_KEY!,
  rules: [
    protectSignup({
      email: {
        mode: isProd ? "LIVE" : "DRY_RUN",
        block: ["DISPOSABLE", "INVALID", "NO_MX_RECORDS"],
      },
      bots: {
        mode: isProd ? "LIVE" : "DRY_RUN",
        allow: [],
      },
      rateLimit: {
        mode: isProd ? "LIVE" : "DRY_RUN",
        interval: "10m",
        max: 5,
      },
    }),
  ],
});
