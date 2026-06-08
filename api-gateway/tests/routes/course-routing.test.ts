import express from "express";
import request from "supertest";
import { setupRoutes } from "../../src/routes/index";
import { mockConfig } from "../helpers/mocks";

const proxyCalls: Array<{ target: any; options: any; req?: any }> = [];

jest.mock("express-http-proxy", () => {
  return jest.fn((target: any, options: any) => {
    const entry: { target: any; options: any; req?: any } = { target, options };
    proxyCalls.push(entry);
    return (req: any, res: any, next: any) => {
      entry.req = {
        method: req.method,
        originalUrl: req.originalUrl,
      };
      res.status(204).end();
    };
  });
});

jest.mock("../../src/services/health.service", () => ({
  checkAllServices: jest.fn().mockResolvedValue({
    status: "ok",
    service: "api-gateway",
    timestamp: new Date().toISOString(),
  }),
}));

describe("Course routing split", () => {
  beforeEach(() => {
    proxyCalls.length = 0;
    jest.clearAllMocks();
  });

  it("routes GET /api/v1/courses?search=python to recommendation-service", async () => {
    const app = express();
    const config = mockConfig();
    setupRoutes(app, config);

    await request(app).get("/api/v1/courses?search=python");

    const matched = proxyCalls.find(call => call.req?.originalUrl === "/api/v1/courses?search=python");
    const target = typeof matched?.target === "function" ? matched?.target() : matched?.target;
    expect(target).toBe(config.services.recommendation[0].url);
  });

  it("keeps GET /api/v1/courses without search on courses-service", async () => {
    const app = express();
    const config = mockConfig();
    setupRoutes(app, config);

    await request(app).get("/api/v1/courses");

    const matched = proxyCalls.find(call => call.req?.originalUrl === "/api/v1/courses");
    const target = typeof matched?.target === "function" ? matched?.target() : matched?.target;
    expect(target).toBe(config.services.courses[0].url);
  });

  it("routes GET /api/v1/courses/autocomplete?search=py to recommendation-service", async () => {
    const app = express();
    const config = mockConfig();
    setupRoutes(app, config);

    await request(app).get("/api/v1/courses/autocomplete?search=py");

    const matched = proxyCalls.find(call => call.req?.originalUrl === "/api/v1/courses/autocomplete?search=py");
    const target = typeof matched?.target === "function" ? matched?.target() : matched?.target;
    expect(target).toBe(config.services.recommendation[0].url);
  });

  it("routes GET /api/v1/courses/analytics/top-query-courses to recommendation-service", async () => {
    const app = express();
    const config = mockConfig();
    setupRoutes(app, config);

    await request(app).get("/api/v1/courses/analytics/top-query-courses");

    const matched = proxyCalls.find(call => call.req?.originalUrl === "/api/v1/courses/analytics/top-query-courses");
    const target = typeof matched?.target === "function" ? matched?.target() : matched?.target;
    expect(target).toBe(config.services.recommendation[0].url);
  });

  it("keeps GET /api/v1/courses/:id on courses-service", async () => {
    const app = express();
    const config = mockConfig();
    setupRoutes(app, config);

    await request(app).get("/api/v1/courses/123");

    const matched = proxyCalls.find(call => call.req?.originalUrl === "/api/v1/courses/123");
    const target = typeof matched?.target === "function" ? matched?.target() : matched?.target;
    expect(target).toBe(config.services.courses[0].url);
  });
});
