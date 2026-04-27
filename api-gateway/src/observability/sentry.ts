import * as Sentry from '@sentry/node';
import { Express } from 'express';
import { trace, context } from '@opentelemetry/api';

export const setupSentry = (app: Express) => {
  const dsn = process.env.SENTRY_DSN;
  if (!dsn) {
    console.warn('[Sentry] SENTRY_DSN not provided, skipping initialization');
    return;
  }

  Sentry.init({
    dsn: dsn,
    environment: process.env.NODE_ENV || 'production',
    tracesSampleRate: 1.0,
  });

  // RequestHandler creates a separate execution context, so that all
  // transactions/spans/breadcrumbs are isolated per request
  app.use(Sentry.Handlers.requestHandler());
  // TracingHandler creates a trace for every incoming request
  app.use(Sentry.Handlers.tracingHandler());

  // Attach trace_id to sentry scope
  app.use((req, res, next) => {
    const span = trace.getSpan(context.active());
    if (span) {
      const traceId = span.spanContext().traceId;
      Sentry.configureScope((scope: any) => {
        scope.setTag('trace_id', traceId);
      });
    }
    next();
  });
};

export const setupSentryErrorHandler = (app: Express) => {
  if (process.env.SENTRY_DSN) {
    app.use(Sentry.Handlers.errorHandler());
  }
};
