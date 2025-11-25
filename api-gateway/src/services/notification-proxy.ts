import express from "express";
import proxy from "express-http-proxy";
import { PROXY_CONFIG } from "../config/index.js";

/**
 * Create proxy middleware for the notification service
 */
export const createNotificationServiceProxy = () => {
  return proxy(PROXY_CONFIG.notificationServiceUrl, {
    https: false,
    proxyReqOptDecorator: (proxyReqOpts: any) => {
      proxyReqOpts.headers = proxyReqOpts.headers || {};
      return proxyReqOpts;
    },
    userResHeaderDecorator: (headers: any) => {
      return headers;
    }
  });
};
