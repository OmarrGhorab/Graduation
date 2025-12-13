import * as admin from "firebase-admin";
import dotenv from "dotenv";

dotenv.config();

// Initialize Firebase Admin SDK
let messaging: admin.messaging.Messaging | null = null;

try {
  // Check if Firebase credentials are provided
  const projectId = process.env.FIREBASE_PROJECT_ID;
  const privateKey = process.env.FIREBASE_PRIVATE_KEY?.replace(/\\n/g, "\n");
  const clientEmail = process.env.FIREBASE_CLIENT_EMAIL;

  if (projectId && privateKey && clientEmail) {
    // Initialize with service account credentials
    if (!admin.apps.length) {
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

