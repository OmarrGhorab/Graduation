// Import tracing first!
import './tracing.js';
import { Express } from 'express';
import { metricsMiddleware, setupMetricsEndpoint } from './metrics.js';
import { instrumentConsole } from './logger.js';
import { setupSentry, setupSentryErrorHandler } from './sentry.js';

export const initObservability = (app: Express) => {
  // Replace console.log with structured logger
  instrumentConsole();

  // Metrics middleware
  app.use(metricsMiddleware);

  // Setup /metrics endpoint
  setupMetricsEndpoint(app);

  // Setup Sentry
  setupSentry(app);

  console.log('[Observability] Initialized');
};

export { setupSentryErrorHandler };
