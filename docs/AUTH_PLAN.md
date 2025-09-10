# Zitadel Authentication with SolidJS and Go

The preferred authentication flow for a SolidJS client and Go backend using Zitadel is the **Authorization Code Flow with Proof Key for Code Exchange (PKCE)**. This is a robust and secure standard for web and mobile applications.

---

### SolidJS Client (Frontend)

Your SolidJS client is a **user-agent** and handles the user-facing authentication. Since it runs in the browser, it cannot securely store a client secret, which is why PKCE is crucial.

1.  **Initiate the Login Flow**: Your SolidJS app generates a `code_verifier` and a `code_challenge`. The `code_challenge` is used to construct the authorization URL.
2.  **Redirect to Zitadel**: The client redirects the user's browser to the Zitadel authorization endpoint with the following parameters:
    - `client_id`: The public ID of your SolidJS application in Zitadel.
    - `redirect_uri`: The URL on your SolidJS app where Zitadel redirects the user after authentication.
    - `response_type`: `code` to request an authorization code.
    - `scope`: The permissions your app needs (e.g., `openid`, `profile`, `email`, and a scope for your backend API).
    - `code_challenge`: The cryptographic challenge generated earlier.
    - `code_challenge_method`: `S256` for the hashing method.
    - `state`: A random, unguessable value to protect against CSRF.

---

### Go Backend (Resource Server)

Your Go backend will be the resource server that protects your APIs. It should be configured to accept and validate tokens issued by Zitadel.

1.  **Create an API Application**: In your Zitadel project, define your Go backend as an **API** application.
2.  **Use a Go SDK**: Zitadel provides an official Go SDK (`github.com/zitadel/zitadel-go`) with convenient middleware and helpers for OIDC.
3.  **Validate Tokens**: When the SolidJS client makes a request to your Go backend, it will include the access token in the `Authorization` header. Your Go backend should:
    - Extract the token.
    - Use the Zitadel Go SDK to validate the token's signature against Zitadel's public keys.
    - Check the token's claims to ensure it's valid for your API and that the user has the necessary permissions.

---

### Token Exchange

After the user successfully authenticates on the Zitadel login page, Zitadel will redirect them back to the `redirect_uri` on your SolidJS client with a temporary `authorization_code`.

1.  **SolidJS Client Receives Code**: Your SolidJS client receives the `authorization_code` and the `state` parameter, and verifies the `state`.
2.  **SolidJS Exchanges the Code**: The SolidJS client makes a direct request to Zitadel's token endpoint to exchange the `authorization_code` for an `ID token` and an `access token`. This request must include the `code_verifier`.
3.  **SolidJS Stores Tokens**: The SolidJS client stores the received tokens securely, typically in memory or local storage, and uses the access token for all future API calls to the Go backend.

---

### Adding Mobile Applications

The beauty of the Authorization Code with PKCE flow is that it is the standard for native (mobile) and web applications. The implementation for a mobile app (iOS, Android, or Flutter) will follow the exact same logic as the SolidJS client:

- The mobile app opens a system browser tab for the user to log in.
- Zitadel redirects back to a custom URI scheme registered by your mobile app.
- The app then uses the received `authorization_code` to exchange for tokens.

---

## Current Implementation Status

### ✅ What's Working
- Basic OIDC endpoints configured (`/auth/login-url`, `/auth/token-exchange`)
- Frontend AuthContext structure in place with state management
- Backend ZitadelService with authorization URL generation
- CSRF protection via state parameter

### ❌ Missing PKCE Implementation
**Frontend Gaps:**
- No `code_verifier` generation (should use crypto.getRandomValues)
- No `code_challenge` creation (SHA256 hash + base64url encoding)
- Not sending `code_challenge` to backend `/auth/login-url`
- Not including `code_verifier` in token exchange request

**Backend Gaps:**
- `GetAuthorizationURL()` doesn't accept `code_challenge` parameter
- Authorization URL missing PKCE parameters (`code_challenge`, `code_challenge_method=S256`)
- Token exchange still trying to use `client_secret` instead of `code_verifier`

### ❌ Backend Token Validation Issues
- Token introspection failing due to missing backend credentials
- Need proper authentication method for backend to validate tokens

---

## PKCE Implementation Details

### Frontend (SolidJS) Changes Needed

**1. Generate PKCE Parameters:**
```typescript
// In AuthContext.tsx - loginWithOIDC()
function generateCodeVerifier(): string {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  return base64urlEncode(array);
}

async function generateCodeChallenge(verifier: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(verifier);
  const digest = await crypto.subtle.digest('SHA-256', data);
  return base64urlEncode(new Uint8Array(digest));
}
```

**2. Update Login Flow:**
- Generate and store `code_verifier` in localStorage
- Create `code_challenge` from verifier
- Send `code_challenge` to backend `/auth/login-url` endpoint
- Include `code_verifier` in token exchange request

### Backend (Go) Changes Needed

**1. Update Authorization URL Generation:**
```go
// ZitadelService.GetAuthorizationURL() should accept code_challenge
func (zs *ZitadelService) GetAuthorizationURL(state, redirectURI, codeChallenge string) string {
    return fmt.Sprintf("%s/oauth/v2/authorize?client_id=%s&response_type=code&redirect_uri=%s&state=%s&scope=openid+profile+email&code_challenge=%s&code_challenge_method=S256",
        strings.TrimSuffix(zs.issuer, "/"),
        zs.clientID,
        redirectURI,
        state,
        codeChallenge,
    )
}
```

**2. Update Token Exchange:**
```go
// TokenExchangeRequest should include code_verifier
type TokenExchangeRequest struct {
    Code         string `json:"code"`
    RedirectURI  string `json:"redirect_uri"`
    State        string `json:"state,omitempty"`
    CodeVerifier string `json:"code_verifier"` // Add this
}
```

---

## Backend Token Validation Strategy

### Current Problem
Backend cannot validate tokens because it lacks proper credentials for Zitadel token introspection.

### Solution Options

**Option A: Machine-to-Machine Application (Recommended)**
1. Create separate Zitadel M2M application for backend
2. Get `client_id` and `client_secret` for backend service
3. Use M2M credentials for token introspection
4. Keep frontend OIDC app as public client (no secret)

**Option B: JWT Token Validation (Alternative)**
1. Implement direct JWT validation using Zitadel's public keys
2. Fetch JWKS from `/.well-known/openid_configuration`
3. Validate token signatures locally
4. No backend credentials needed
5. Better performance (no API calls)

---

## Implementation Roadmap

### Phase 1: Complete PKCE Flow
**Priority: High** - Security requirement for public clients
1. Frontend: Implement PKCE parameter generation
2. Backend: Update endpoints to handle PKCE parameters
3. Test: Authorization flow with proper PKCE validation

### Phase 2: Fix Backend Token Validation
**Priority: High** - Required for protected endpoints
1. Choose validation strategy (M2M app vs JWT validation)
2. Implement chosen approach
3. Test: Token validation for protected API calls

### Phase 3: End-to-End Testing
**Priority: Medium** - Ensure complete flow works
1. Test full authentication flow
2. Test protected API endpoints
3. Test token refresh (if implemented)
4. Test error scenarios

---

## Files Requiring Updates

### Frontend
- `client/src/context/AuthContext.tsx` - Add PKCE generation
- `client/src/types/User.ts` - Update request types

### Backend  
- `server/internal/services/zitadel.service.go` - Add PKCE support
- `server/internal/handlers/auth.handler.go` - Update endpoints
- `server/config/config.go` - Add M2M credentials (if chosen)

### Documentation
- Update this plan with implementation progress
- Document chosen token validation approach
