/**
 * Simple debug logger utility
 */

export function debugLog(message: string, ...args: any[]): void {
  if (process.env.NODE_ENV === 'development') {
    console.log(`[DEBUG] ${message}`, ...args);
  }
}
