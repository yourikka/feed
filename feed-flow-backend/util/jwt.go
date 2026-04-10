package util

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 签名密钥 (实际开发请放环境变量，不要硬编码)
var jwtSecret = []byte(getJWTSecret())

// Claims 自定义Token结构
type Claims struct {
	UserID uint // 存用户ID
	jwt.RegisteredClaims
}

// GenerateToken 生成Token
func GenerateToken(userID uint) (string, error) {
	now := time.Now()
	expire := now.Add(24 * time.Hour) // Token 24小时后过期

	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expire),
			Issuer:    "douyin-demo",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken 解析Token
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func getJWTSecret() string {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return secret
	}
	return "my_douyin_secret_key"
}
