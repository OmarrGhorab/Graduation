// Barrel export for all auth controllers
// This file exports all auth-related controllers for centralized imports

// Core authentication controllers
export {
    registerUser,
    loginUser,
    logoutUser,
    refreshToken,
} from "./auth.core.controller";

// Password management controllers
export {
    forgotPassword,
    resetPassword,
} from "./password.controller";

// Email verification controllers
export {
    resendVerificationOtp,
    verifyEmailOtp,
} from "./email-verification.controller";

// OAuth controllers
export {
    googleMobileAuth,
} from "./oauth.controller";

// Device verification controllers
export {
    verifyDevice,
    resendDeviceVerificationOtp,
} from "./device.controller";
