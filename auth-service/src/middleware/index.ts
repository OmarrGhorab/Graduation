export { errorHandler } from "./errorHandler";
export { authenticate, requireRole } from "./auth.middleware";
export { authenticateDeactivated } from "./authenticateDeactivated.middleware";
export { authenticateInternalService } from "./internal-auth.middleware";
export { 
  extractDeviceInfo, 
  getDeviceInfoFromRequest, 
  getDeviceLocationFromRequest,
  hasValidLocation 
} from "./deviceInfo.middleware";
export type { DeviceInfo, DeviceLocationData } from "./deviceInfo.middleware";

