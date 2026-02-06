# Authentication Package

This package provides authentication and authorization functionality for the AI Research Platform.

## Components

### Password Hashing (`password.go`)

- **PasswordHasher**: Handles secure password hashing using bcrypt
- **HashPassword**: Hashes plaintext passwords with configurable cost
- **VerifyPassword**: Verifies passwords against hashed values
- **IsPasswordHashed**: Checks if a string is a valid bcrypt hash

### JWT Management (`jwt.go`)

- **JWTManager**: Manages JWT token generation and validation
- **GenerateToken**: Creates JWT tokens with user identity claims
- **ValidateToken**: Validates JWT tokens and extracts claims
- **RefreshToken**: Generates new tokens with extended expiration
- **Claims**: JWT claims structure with user information

## Usage

### Password Hashing

```go
hasher := auth.NewPasswordHasher(12) // bcrypt cost of 12

// Hash a password
hashed, err := hasher.HashPassword("mySecurePassword")
if err != nil {
    // Handle error
}

// Verify a password
err = hasher.VerifyPassword(hashed, "mySecurePassword")
if err != nil {
    // Password doesn't match
}
```

### JWT Tokens

```go
jwtManager := auth.NewJWTManager("secret-key", 24*time.Hour)

// Generate a token
token, err := jwtManager.GenerateToken("user-123", "user@example.com", "username")
if err != nil {
    // Handle error
}

// Validate a token
claims, err := jwtManager.ValidateToken(token)
if err != nil {
    // Invalid or expired token
}

// Access user information
userID := claims.UserID
email := claims.Email
username := claims.Username
```

## Testing

The package includes comprehensive property-based tests using gopter:

- **Property 34**: Password hashing security (Requirements 12.1)
- **Property 35**: JWT token issuance (Requirements 12.2)
- **Property 36**: JWT token validation (Requirements 12.3)

Run tests:
```bash
go test ./internal/auth/...
```

## Security Considerations

- Passwords are hashed using bcrypt with a minimum cost of 10 (12 recommended for production)
- JWT tokens use HS256 signing algorithm
- Tokens include expiration, issued at, and not before claims
- All sensitive operations are tested with property-based testing
