package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher handles password hashing and verification
type PasswordHasher struct {
	cost int
}

// NewPasswordHasher creates a new password hasher with the specified bcrypt cost
func NewPasswordHasher(cost int) *PasswordHasher {
	return &PasswordHasher{
		cost: cost,
	}
}

// HashPassword hashes a plaintext password using bcrypt
func (h *PasswordHasher) HashPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashedBytes), nil
}

// VerifyPassword checks if a plaintext password matches a hashed password
func (h *PasswordHasher) VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// IsPasswordHashed checks if a string is a valid bcrypt hash
func IsPasswordHashed(password string) bool {
	// Bcrypt hashes start with $2a$, $2b$, or $2y$ and are 60 characters long
	if len(password) != 60 {
		return false
	}
	
	if len(password) < 4 {
		return false
	}
	
	prefix := password[:4]
	return prefix == "$2a$" || prefix == "$2b$" || prefix == "$2y$"
}

// Default password hasher with cost 12
var defaultHasher = NewPasswordHasher(12)

// HashPassword is a convenience function that hashes a password using the default hasher
func HashPassword(password string) (string, error) {
	return defaultHasher.HashPassword(password)
}

// CheckPassword is a convenience function that verifies a password against a hash
func CheckPassword(password, hashedPassword string) bool {
	err := defaultHasher.VerifyPassword(hashedPassword, password)
	return err == nil
}
