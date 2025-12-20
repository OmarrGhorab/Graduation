import admin from "firebase-admin";
import dotenv from "dotenv";
import path from "path";
import { fileURLToPath } from "url";

// ES module equivalent of __dirname
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Load .env from the project root (notification-service folder)
const envPath = path.resolve(__dirname, "../../.env");
console.log("[Firebase] Loading .env from:", envPath);
const result = dotenv.config({ path: envPath });
if (result.error) {
  console.error("[Firebase] Error loading .env:", result.error);
} else {
  console.log("[Firebase] .env loaded successfully");
}

// Initialize Firebase Admin SDK
let messaging: admin.messaging.Messaging | null = null;

try {
  // Check if Firebase credentials are provided
  const projectId = process.env.FIREBASE_PROJECT_ID;
  const privateKeyRaw = process.env.FIREBASE_PRIVATE_KEY;
  const privateKey = privateKeyRaw ? privateKeyRaw.replace(/\\n/g, "\n") : undefined;
  const clientEmail = process.env.FIREBASE_CLIENT_EMAIL;



  if (projectId && privateKey && clientEmail) {
    // Initialize with service account credentials
    if (!admin.apps || admin.apps.length === 0) {
      admin.initializeApp({
        credential: admin.credential.cert({
          projectId,
          privateKey,
          clientEmail,
        }),
      });
    }

    messaging = admin.messaging();
    console.log("Firebase Admin SDK initialized successfully");
  } else {
    console.warn(
      "Firebase credentials not found. FCM push notifications will be disabled. " +
      "Set FIREBASE_PROJECT_ID, FIREBASE_PRIVATE_KEY, and FIREBASE_CLIENT_EMAIL environment variables."
    );
  }
} catch (error) {
  console.error("Error initializing Firebase Admin SDK:", error);
  console.warn("FCM push notifications will be disabled");
  
}

export { messaging };
export default admin;

