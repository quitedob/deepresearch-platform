package auth

import (
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

// Claims JWT声明
type Claims struct {
    UserID   string `json:"user_id"`
    Email    string `json:"email"`
    Username string `json:"username"`
    jwt.RegisteredClaims
}

// JWTManager JWT管理器
type JWTManager struct {
    secretKey  string
    expiration time.Duration
}

// NewJWTManager 创建JWT管理器
func NewJWTManager(secretKey string, expiration time.Duration) *JWTManager {
    return &JWTManager{
        secretKey:  secretKey,
        expiration: expiration,
    }
}

// GenerateToken 生成JWT令牌
func (m *JWTManager) GenerateToken(userID, email, username string) (string, error) {
    if userID == "" {
        return "", fmt.Errorf("user ID cannot be empty")
    }

    now := time.Now()
    claims := Claims{
        UserID:   userID,
        Email:    email,
        Username: username,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(now.Add(m.expiration)),
            IssuedAt:  jwt.NewNumericDate(now),
            NotBefore: jwt.NewNumericDate(now),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(m.secretKey))
}

// ValidateToken 验证JWT令牌
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(m.secretKey), nil
    })

    if err != nil {
        return nil, err
    }

    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid token")
    }

    return claims, nil
}

// RefreshToken 刷新JWT令牌
func (m *JWTManager) RefreshToken(tokenString string) (string, error) {
    claims, err := m.ValidateToken(tokenString)
    if err != nil {
        return "", fmt.Errorf("invalid token: %v", err)
    }

    // 生成新令牌
    return m.GenerateToken(claims.UserID, claims.Email, claims.Username)
}
