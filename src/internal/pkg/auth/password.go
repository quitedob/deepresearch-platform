package auth

import (
    "fmt"
    "golang.org/x/crypto/bcrypt"
)

// PasswordHasher 密码哈希器
type PasswordHasher struct {
    cost int
}

// NewPasswordHasher 创建密码哈希器
func NewPasswordHasher(cost int) *PasswordHasher {
    return &PasswordHasher{
        cost: cost,
    }
}

// HashPassword 哈希密码
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

// VerifyPassword 验证密码
func (h *PasswordHasher) VerifyPassword(hashedPassword, password string) error {
    return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// IsPasswordHashed 检查密码是否已哈希
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

// 默认密码哈希器，cost=12
var defaultHasher = NewPasswordHasher(12)

// HashPassword 使用默认哈希器哈希密码
func HashPassword(password string) (string, error) {
    return defaultHasher.HashPassword(password)
}

// CheckPassword 使用默认哈希器验证密码
func CheckPassword(password, hashedPassword string) bool {
    err := defaultHasher.VerifyPassword(hashedPassword, password)
    return err == nil
}
