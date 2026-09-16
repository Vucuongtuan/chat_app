# JWT Package

The `jwt` package handles JSON Web Token (JWT) creation, validation, and parsing. It provides utilities for secure token-based authentication.

## Files

### `jwt.go`
Implements JWT token operations.

**Functions:**
- `GenerateToken(user *model.User, secret string, expiration time.Duration) (string, error)` - Create JWT token
- `GenerateRefreshToken(userID string, secret string, expiration time.Duration) (string, error)` - Create refresh token
- `ValidateToken(tokenString, secret string) (*Claims, error)` - Validate and parse token
- `RefreshToken(refreshToken, secret string, newExpiration time.Duration) (string, error)` - Issue new access token

**Responsibilities:**
- Create signed JWT tokens
- Validate token signatures
- Parse token claims
- Handle token expiration
- Manage token refresh

## Token Structure

### Access Token Claims

```go
type Claims struct {
    UserID   string `json:"user_id"`
    Email    string `json:"email"`
    Username string `json:"username"`
    jwt.RegisteredClaims
}

type RegisteredClaims struct {
    ExpiresAt *time.Time `json:"exp"`
    IssuedAt  *time.Time `json:"iat"`
    NotBefore *time.Time `json:"nbf"`
    Subject   string     `json:"sub"`
}
```

### Token Payload Example

```json
{
  "user_id": "uuid-123",
  "email": "user@example.com",
  "username": "johndoe",
  "exp": 1694529600,
  "iat": 1694443200,
  "nbf": 1694443200
}
```

## Token Generation

**Access Token:**
- Lifetime: 1 hour (configurable)
- Used for API request authentication
- Included in every authenticated request

**Refresh Token:**
- Lifetime: 7 days (configurable)
- Used to obtain new access token
- Stored securely in client

```go
accessToken, err := jwt.GenerateToken(user, secret, 1*time.Hour)
refreshToken, err := jwt.GenerateRefreshToken(user.ID, secret, 7*24*time.Hour)
```

## Token Validation

Validates token signature and expiration:

```go
claims, err := jwt.ValidateToken(tokenString, secret)
if err != nil {
    // Token invalid or expired
    return err
}

// Access claims
userID := claims.UserID
email := claims.Email
```

## Refresh Token Flow

```
1. User logs in → Server generates access + refresh tokens
2. Client stores both tokens
3. Client uses access token in Authorization header
4. Access token expires
5. Client uses refresh token to get new access token
6. Server validates refresh token, issues new access token
7. If refresh token expired, user must login again
```

## Configuration

JWT configuration from environment:

```go
type JWTConfig struct {
    Secret string        // JWT_SECRET
    Expire time.Duration // JWT_EXPIRE (e.g., "24h")
}
```

## Usage in Authentication Flow

### Login Response
```go
func Login(user *model.User, secret string) AuthTokenResponse {
    accessToken, _ := jwt.GenerateToken(user, secret, 1*time.Hour)
    refreshToken, _ := jwt.GenerateRefreshToken(user.ID, secret, 7*24*time.Hour)
    
    return AuthTokenResponse{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        ExpiresIn:    3600, // seconds
        TokenType:    "Bearer",
    }
}
```

### Verify Request
```go
authHeader := c.GetHeader("Authorization")
token := strings.TrimPrefix(authHeader, "Bearer ")

claims, err := jwt.ValidateToken(token, secret)
if err != nil {
    c.JSON(401, gin.H{"error": "invalid_token"})
    c.Abort()
    return
}

c.Set("user_id", claims.UserID)
```

### Refresh Token
```go
func RefreshAccessToken(refreshToken string, secret string) (string, error) {
    // Validate refresh token
    claims, err := jwt.ValidateToken(refreshToken, secret)
    if err != nil {
        return "", err
    }
    
    // Generate new access token
    user := &model.User{ID: claims.UserID}
    return jwt.GenerateToken(user, secret, 1*time.Hour)
}
```

## Security Best Practices

1. **Secret Management**
   - Store secret in environment variable
   - Use strong, random secret (32+ characters)
   - Never hardcode secret in code

2. **Token Storage**
   - Access token: Memory or httpOnly cookie
   - Refresh token: httpOnly secure cookie
   - Never store in localStorage

3. **Token Transmission**
   - Always use HTTPS/TLS
   - Send in Authorization header: `Bearer <token>`
   - Never log tokens

4. **Token Expiration**
   - Short-lived access tokens (1 hour)
   - Longer-lived refresh tokens (7 days)
   - Implement token rotation

## Common Errors

- `signature is invalid` - Token tampered or wrong secret
- `token is expired` - Token lifetime exceeded
- `token used before valid` - Token nbf claim not reached
- `invalid claims` - Missing required claims

## Testing

Mock JWT for testing:

```go
func MockToken(userID string) string {
    claims := &jwt.Claims{
        UserID: userID,
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, _ := token.SignedString([]byte("test-secret"))
    return tokenString
}
```
