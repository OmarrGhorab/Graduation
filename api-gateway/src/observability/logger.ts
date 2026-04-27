import pino from 'pino';
import { trace, context } from '@opentelemetry/api';

export const logger = pino({
  level: process.env.LOG_LEVEL || 'info',
  mixin() {
    const span = trace.getSpan(context.active());
    if (span) {
      const spanContext = span.spanContext();
      return {
        trace_id: spanContext.traceId,
        span_id: spanContext.spanId,
        service_name: process.env.OTEL_SERVICE_NAME || 'api-gateway',
      };
    }
    return {
      service_name: process.env.OTEL_SERVICE_NAME || 'api-gateway',
    };
  },
});

// Replaces console.log with structured logging
export const instrumentConsole = () => {
  console.log = (...args) => logger.info(args.length > 1 ? args : args[0]);
  console.info = (...args) => logger.info(args.length > 1 ? args : args[0]);
  console.error = (...args) => logger.error(args.length > 1 ? args : args[0]);
  console.warn = (...args) => logger.warn(args.length > 1 ? args : args[0]);
};
