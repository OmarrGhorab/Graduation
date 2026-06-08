import * as fc from "fast-check";
import express, { Express } from "express";
import { setupRoutes } from "../../src/routes/index";
import { mockConfig } from "../helpers/mocks";
import { AppConfig } from "../../src/config/index";

jest.mock("express-http-proxy", () => {
  return jest.fn((_target: any, _options: any) => {
    return (_req: any, _res: any, next: any) => {
      next();
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

function resolveTarget(target: any): string {
  return typeof target === "function" ? target() : target;
}

describe("Proxy Routes", () => {
  let app: Express;
  let config: AppConfig;
  let proxy: jest.Mock;

  beforeEach(() => {
    jest.clearAllMocks();
    app = express();
    config = mockConfig();
    proxy = require("express-http-proxy") as jest.Mock;
  });

  it("registers notification, courses, payment, recommendation, chat, reports, auth, and course search proxies", () => {
    setupRoutes(app, config);

    expect(proxy.mock.calls.length).toBeGreaterThan(10);
    const targets = proxy.mock.calls.map(call => resolveTarget(call[0]));

    expect(targets).toContain(config.services.notification[0].url);
    expect(targets).toContain(config.services.courses[0].url);
    expect(targets).toContain(config.services.payment[0].url);
    expect(targets).toContain(config.services.recommendation[0].url);
    expect(targets).toContain(config.services.auth[0].url);
  });

  it("configures path preservation for every proxy", () => {
    setupRoutes(app, config);

    for (const call of proxy.mock.calls) {
      expect(call[1]).toHaveProperty("proxyReqPathResolver");
      expect(typeof call[1].proxyReqPathResolver).toBe("function");
      const requestPath = "/api/v1/courses?search=python";
      expect(call[1].proxyReqPathResolver({ originalUrl: requestPath })).toBe(requestPath);
    }
  });

  it("does not proxy the /health endpoint", () => {
    setupRoutes(app, config);

    const targets = proxy.mock.calls.map(call => resolveTarget(call[0]));
    expect(targets).not.toContain("/health");
  });

  it("preserves any request path when proxying upstream", () => {
    fc.assert(
      fc.property(
        fc.oneof(
          fc.constantFrom(
            "/api/v1/notifications",
            "/api/v1/courses?search=data",
            "/api/v1/courses/autocomplete?search=py",
            "/api/v1/recommendations",
            "/api/v1/auth/login"
          ),
          fc.string({ minLength: 1, maxLength: 40 }).map(value => `/${value}`)
        ),
        (requestPath) => {
          proxy.mockClear();
          app = express();
          setupRoutes(app, config);

          for (const call of proxy.mock.calls) {
            const resolver = call[1].proxyReqPathResolver;
            expect(resolver({ originalUrl: requestPath })).toBe(requestPath);
          }
        }
      ),
      { numRuns: 50 }
    );
  });
});
