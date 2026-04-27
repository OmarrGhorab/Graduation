import client from 'prom-client';
import { Request, Response, NextFunction, Express } from 'express';

// Enable default metrics
client.collectDefaultMetrics();

// Custom metrics
const httpRequestsTotal = new client.Counter({
  name: 'http_requests_total',
  help: 'Total number of HTTP requests',
  labelNames: ['method', 'route', 'status_code'],
});

const httpRequestDurationSeconds = new client.Histogram({
  name: 'http_request_duration_seconds',
  help: 'Duration of HTTP requests in seconds',
  labelNames: ['method', 'route', 'status_code'],
  buckets: [0.1, 0.3, 0.5, 0.7, 1, 3, 5, 7, 10],
});

export const metricsMiddleware = (req: Request, res: Response, next: NextFunction) => {
  const start = process.hrtime();

  res.on('finish', () => {
    const duration = process.hrtime(start);
    const durationInSeconds = duration[0] + duration[1] / 1e9;
    const route = req.route ? req.route.path : req.path;

    httpRequestsTotal.inc({
      method: req.method,
      route: route,
      status_code: res.statusCode,
    });

    httpRequestDurationSeconds.observe(
      {
        method: req.method,
        route: route,
        status_code: res.statusCode,
      },
      durationInSeconds
    );
  });

  next();
};

export const setupMetricsEndpoint = (app: Express) => {
  app.get('/metrics', async (req: Request, res: Response) => {
    res.set('Content-Type', client.register.contentType);
    res.end(await client.register.metrics());
  });
};
