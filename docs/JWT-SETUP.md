# JWT Authentication Setup Guide

This guide covers JWT (JSON Web Token) authentication setup for the Logistics Application.

## Table of Contents
1. [Overview](#overview)
2. [Generating JWT Secret](#generating-jwt-secret)
3. [Configuration](#configuration)
4. [JWT Manager Usage](#jwt-manager-usage)
5. [Security Best Practices](#security-best-practices)
6. [Testing](#testing)

---

## Overview

We use JWT for stateless authentication in our API. Key features:

- **Algorithm**: HMAC-SHA256 (HS256)
- **Library**: `github.com/golang-jwt/jwt/v5`
- **Token Expiry**: Configurable (default: 7 days)
- **Claims**: User ID, Email, Role, Standard JWT claims

### Why JWT?

1. **Stateless**: No session storage needed
2. **Scalable**: Works across multiple servers
3. **Secure**: Cryptographically signed
4. **Standard**: Industry-standard authentication method

---

## Generating JWT Secret

### Production Environment

**CRITICAL**: Never use default or weak secrets in production!

Generate a cryptographically secure secret:

```bash
# Using our key generator
go run cmd/keygen/main.go

# Or using OpenSSL
openssl rand -base64 32

# Or using PowerShell (Windows)
[Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Minimum 0 -Maximum 256 }))
```

### Requirements

- Minimum 32 characters (256 bits recommended)
- Use cryptographically secure random generation
- Never commit to version control
- Rotate regularly in production

---

## Configuration

### Environment Variables

Add to your `.env` file:

```env
# JWT Configuration
JWT_SECRET=your-generated-secret-key-here
JWT_EXPIRES_IN=168h  # 7 days
```

### Token Expiry Options

```env
JWT_EXPIRES_IN=1h      # 1 hour
JWT_EXPIRES_IN=24h     # 1 day
JWT_EXPIRES_IN=168h    # 7 days (default)
JWT_EXPIRES_IN=720h    # 30 days
```

**Recommendation**: 
- API tokens: 1-7 days
- Refresh tokens: 30-90 days (implement separate refresh logic)
- Mobile apps: 7-30 days

---

## JWT Manager Usage

### Initialize JWT Manager

```go
import (
    "logistic-app-go/pkg/auth"
    "logistic-app-go/pkg/config"
)

// Load configuration
cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}

// Create JWT manager
jwtManager, err := auth.NewJWTManager(
    cfg.JWT.Secret,
    cfg.JWT.ExpiresIn,
)
if err != nil {
    log.Fatal(err)
}
```

### Generate Token (Login)

```go
// After successful authentication
token, err := jwtManager.Generate(
    user.ID,      // uint: User ID from database
    user.Email,   // string: User email
    user.Role,    // string: "admin", "dispatcher", "driver", "customer"
)
if err != nil {
    // Handle error
}

// Return token to client
c.JSON(200, gin.H{
    "token": token,
    "type": "Bearer",
    "expires_in": 604800, // 7 days in seconds
})
```

### Verify Token (Protected Routes)

```go
// Extract token from Authorization header
authHeader := c.GetHeader("Authorization")
tokenString := strings.TrimPrefix(authHeader, "Bearer ")

// Verify and parse token
claims, err := jwtManager.Verify(tokenString)
if err != nil {
    c.JSON(401, gin.H{"error": "Invalid or expired token"})
    return
}

// Access user information
userID := claims.UserID
email := claims.Email
role := claims.Role
```

### Refresh Token

```go
// Get old token from request
oldToken := c.GetHeader("Authorization")
oldToken = strings.TrimPrefix(oldToken, "Bearer ")

// Generate new token
newToken, err := jwtManager.Refresh(oldToken)
if err != nil {
    c.JSON(401, gin.H{"error": "Cannot refresh token"})
    return
}

c.JSON(200, gin.H{
    "token": newToken,
})
```

---

## JWT Claims Structure

Our tokens include these claims:

```go
type Claims struct {
    UserID uint   `json:"user_id"`  // Database user ID
    Email  string `json:"email"`    // User email
    Role   string `json:"role"`     // User role
    
    // Standard JWT claims
    jwt.RegisteredClaims
}
```

### Standard Claims

- `exp` (Expiration Time): When token expires
- `iat` (Issued At): When token was created
- `nbf` (Not Before): Token not valid before this time

### Decoded Token Example

```json
{
  "user_id": 123,
  "email": "john@trucking.com",
  "role": "driver",
  "exp": 1723456789,
  "iat": 1722851989,
  "nbf": 1722851989
}
```

---

## Security Best Practices

### Secret Key Management

1. **Never hardcode secrets** in source code
2. **Use environment variables** for configuration
3. **Rotate secrets regularly** in production
4. **Use different secrets** for dev/staging/production
5. **Store securely** (AWS Secrets Manager, Azure Key Vault, etc.)

### Token Security

1. **HTTPS Only**: Always use HTTPS in production
2. **Short Expiry**: Use reasonable token expiration times
3. **Secure Storage**: 
   - Web: HttpOnly cookies (preferred)
   - Mobile: Secure storage (Keychain/KeyStore)
   - Never: LocalStorage on web
4. **Token Refresh**: Implement refresh token pattern
5. **Revocation**: Consider token blacklist for logout

### Implementation Checklist

- [ ] Strong JWT secret (32+ characters)
- [ ] HTTPS enforced in production
- [ ] Token expiry configured appropriately
- [ ] Secure token storage on client
- [ ] Proper error handling (don't leak info)
- [ ] Rate limiting on auth endpoints
- [ ] Token refresh mechanism
- [ ] Logout/revocation strategy

---

## Testing

### Health Check

Test JWT functionality:

```bash
go run cmd/check/main.go
```

Expected output:
```
🔐 Checking JWT Configuration...
  ✓ JWT secret key validated (44 chars)
  ✓ Token generation working
  ✓ Token verification working
  ✓ Claims validation working
  ✓ Token expiry: 168h0m0s
```

### Manual Testing

#### 1. Generate Test Token

```bash
# Create test program
cat > test_jwt.go << 'EOF'
package main

import (
    "fmt"
    "logistic-app-go/pkg/auth"
    "logistic-app-go/pkg/config"
)

func main() {
    cfg, _ := config.Load()
    jwtManager, _ := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.ExpiresIn)
    
    token, _ := jwtManager.Generate(1, "test@example.com", "admin")
    fmt.Println("Token:", token)
}
EOF

go run test_jwt.go
```

#### 2. Decode Token

Visit [jwt.io](https://jwt.io) and paste your token to inspect claims.

**Note**: Never paste production tokens on public sites!

#### 3. Test API Endpoint (Phase 1+)

```bash
# Login to get token
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'

# Use token for protected endpoint
curl http://localhost:8080/api/users/profile \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

---

## Common Issues

### Issue: "JWT secret key must be at least 32 characters long"

**Solution**: Generate a proper secret key:
```bash
go run cmd/keygen/main.go
```

### Issue: "Invalid or expired token"

**Causes**:
1. Token expired (check `exp` claim)
2. Secret key changed (invalidates all tokens)
3. Token modified/corrupted
4. Clock skew between servers

**Solution**:
- Check token expiry time
- Verify JWT_SECRET matches what generated the token
- Use NTP to synchronize server clocks

### Issue: Token works in dev but not production

**Causes**:
1. Different JWT_SECRET values
2. CORS issues
3. HTTPS/HTTP mismatch

**Solution**:
- Ensure production uses correct secret
- Configure CORS middleware
- Use HTTPS in production

---

## Next Steps

After JWT setup:

1. **Implement Authentication** (Phase 1):
   - User registration endpoint
   - Login endpoint
   - Password hashing with bcrypt
   
2. **Add Middleware** (Phase 1):
   - JWT authentication middleware
   - Role-based authorization
   
3. **Protected Routes** (Phase 1+):
   - Apply middleware to protected endpoints
   - Role-specific route guards

---

## Related Documentation

- [Architecture Overview](ARCHITECTURE.md)
- [Developer Guide](DEVELOPER-GUIDE.md)
- [Security Best Practices](../CONTRIBUTING.md#security)
- [Phase 1 Implementation](../IMPLEMENTATION-PLAN.md#phase-1)

---

**Security Warning**: JWT tokens are credentials. Treat them like passwords:
- Never log tokens
- Never commit tokens to version control
- Never share tokens publicly
- Always use HTTPS in production
- Implement proper token storage and rotation
