# Mobile Backend Cleanup Guide

## Overview

This document identifies code and dependencies in the auth service that are **not needed** for a mobile-only backend. Mobile apps use:
- **Authorization headers** (Bearer tokens) instead of cookies
- **No CORS** (mobile apps don't have origins)
- **Token-based auth** (no cookie management)

---

## 🗑️ Code That Can Be Removed/Simplified

### 1. **Cookie Functions (DEPRECATED but still used)**

**Files:**
- `src/utils/cookies.ts` - Entire file can be simplified

**Current Usage:**
- `setAuthCookies()` - Called in 3 places (twoFactor, device, auth controllers)
- `clearAuthCookies()` - Called in 3 places (sessions, account controllers)
- `parseCookiesFromRequest()` - Used as fallback in token extraction

**Action:** Remove cookie setting/clearing, keep only token extraction from headers

---

### 2. **Cookie Fallback Logic**

**Files:**
- `src/utils/cookies.ts` - `getAccessTokenFromRequest()` and `getRefreshTokenFromRequest()`
- `src/middleware/auth.middleware.ts` - Uses cookie fallback

**Current Code:**
```typescript
// Fallback: Check cookies (for backward compatibility with web clients)
const cookies = parseCookiesFromRequest(req);
return cookies.access_token;
```

**Action:** Remove cookie fallback, use only Authorization header

---

### 3. **CORS Configuration (Can be simplified)**

**File:** `src/main.ts`

**Current:**
```typescript
app.use(cors({
    origin: ["http://localhost:3000", "http://localhost:8080"],
    credentials: false, // ✅ Good - already mobile-friendly
    allowedHeaders: ["Content-Type", "Authorization", "x-refresh-token"],
}));
```

**For Mobile-Only:**
- `origin` restriction not needed (mobile apps don't send origin)
- Can use wildcard or remove origin check entirely

---

### 4. **Session Device Info (Location/IP tracking)**

**Files:**
- `src/utils/sessions.ts` - `getSessionDeviceInfo()`
- Used in all login/register flows

**Current:** Tracks IP address, user agent, location (for security)

**For Mobile:**
- **Keep:** User agent, IP address (useful for security)
- **Optional:** Location tracking (can be removed if not needed)

---

## 📋 Detailed Breakdown

### Cookie Functions Still Being Called

#### `setAuthCookies()` - Called in:
1. `src/controllers/twoFactor.controller.ts:383`
2. `src/controllers/device.controller.ts:125, 158`

#### `clearAuthCookies()` - Called in:
1. `src/controllers/sessions.controller.ts:122, 179`
2. `src/controllers/account.controller.ts:55, 127`

**Impact:** These calls do nothing harmful but add unnecessary code. Mobile apps ignore cookies.

---

## ✅ Recommended Changes

### Option 1: Remove Cookie Functions Entirely (Recommended)

**Changes:**

1. **Remove cookie setting/clearing calls:**
   - Remove all `setAuthCookies()` calls
   - Remove all `clearAuthCookies()` calls
   - Tokens are already returned in response body ✅

2. **Simplify token extraction:**
   ```typescript
   // BEFORE (with cookie fallback)
   export function getAccessTokenFromRequest(req: Request): string | undefined {
     const authHeader = req.headers.authorization;
     if (authHeader && authHeader.startsWith("Bearer ")) {
       return authHeader.substring(7);
     }
     const cookies = parseCookiesFromRequest(req);
     return cookies.access_token;
   }
   
   // AFTER (mobile-only)
   export function getAccessTokenFromRequest(req: Request): string | undefined {
     const authHeader = req.headers.authorization;
     if (authHeader && authHeader.startsWith("Bearer ")) {
       return authHeader.substring(7);
     }
     return undefined;
   }
   ```

3. **Simplify CORS:**
   ```typescript
   // BEFORE
   app.use(cors({
       origin: ["http://localhost:3000", "http://localhost:8080"],
       credentials: false,
       allowedHeaders: ["Content-Type", "Authorization", "x-refresh-token"],
   }));
   
   // AFTER (mobile-only)
   app.use(cors({
       origin: true, // Allow all origins (mobile apps don't send origin)
       credentials: false,
       allowedHeaders: ["Content-Type", "Authorization", "x-refresh-token"],
   }));
   ```

4. **Remove unused cookie parsing:**
   - Remove `parseCookiesFromRequest()` function
   - Remove `setAuthCookies()` function
   - Remove `clearAuthCookies()` function

---

### Option 2: Keep for Backward Compatibility (If supporting web clients)

If you might add a web client later, keep the cookie fallback but:
- Remove all `setAuthCookies()` calls (tokens in body is enough)
- Keep cookie parsing as fallback only
- Document that cookies are deprecated

---

## 📊 Files to Modify

### High Priority (Remove Cookie Calls)

1. ✅ `src/controllers/twoFactor.controller.ts` - Remove `setAuthCookies()` call
2. ✅ `src/controllers/device.controller.ts` - Remove `setAuthCookies()` calls (2 places)
3. ✅ `src/controllers/sessions.controller.ts` - Remove `clearAuthCookies()` calls (2 places)
4. ✅ `src/controllers/account.controller.ts` - Remove `clearAuthCookies()` calls (2 places)

### Medium Priority (Simplify Token Extraction)

5. ✅ `src/utils/cookies.ts` - Remove cookie parsing, simplify token extraction
6. ✅ `src/middleware/auth.middleware.ts` - Remove cookie fallback

### Low Priority (Simplify CORS)

7. ✅ `src/main.ts` - Simplify CORS config (optional)

---

## 🔍 What to Keep

### ✅ Keep These (Mobile-Friendly)

1. **Token-based authentication** ✅
   - Authorization header extraction
   - x-refresh-token header
   - Token validation

2. **Session management** ✅
   - Database sessions (useful for security)
   - Session revocation
   - Multiple device support

3. **Device tracking** ✅
   - Device fingerprinting
   - User agent tracking
   - IP address logging (for security)

4. **Response body tokens** ✅
   - Tokens returned in JSON response
   - Mobile apps store in secure storage

---

## 📝 Code Examples

### Before (With Cookie Support)

```typescript
// Login response
res.json({
  accessToken,
  refreshToken,
  user: { id, email, username }
});
setAuthCookies(res, accessToken, refreshToken); // ❌ Not needed for mobile
```

### After (Mobile-Only)

```typescript
// Login response
res.json({
  accessToken,
  refreshToken,
  user: { id, email, username }
});
// ✅ No cookie setting needed
```

---

## 🎯 Summary

### Can Remove:
- ❌ `setAuthCookies()` function and all calls
- ❌ `clearAuthCookies()` function and all calls  
- ❌ `parseCookiesFromRequest()` function
- ❌ Cookie fallback in token extraction
- ❌ CORS origin restrictions (optional)

### Must Keep:
- ✅ Authorization header extraction
- ✅ Token validation
- ✅ Session management (database)
- ✅ Device tracking
- ✅ Response body tokens

### Impact:
- **Code reduction:** ~100-150 lines
- **Simpler code:** Less branching logic
- **Mobile-first:** Cleaner API for mobile clients
- **No breaking changes:** Mobile apps already use headers

---

## 🚀 Migration Steps

1. Remove all `setAuthCookies()` calls (6 places)
2. Remove all `clearAuthCookies()` calls (4 places)
3. Simplify `getAccessTokenFromRequest()` - remove cookie fallback
4. Simplify `getRefreshTokenFromRequest()` - remove cookie fallback
5. Remove unused cookie parsing functions
6. (Optional) Simplify CORS configuration
7. Test all authentication flows
8. Verify tokens still work via Authorization header

---

## ⚠️ Important Notes

- **Tokens are already returned in response body** ✅
- **Mobile apps already use Authorization headers** ✅
- **No breaking changes** - cookies were just a fallback
- **Sessions still work** - they're stored in database, not cookies
- **Security maintained** - token validation unchanged

