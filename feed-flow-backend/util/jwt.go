package util

import (
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtSecretOnce      sync.Once
	jwtSecret          []byte
	jwtSecretErr       error
	snapshotSecretOnce sync.Once
	snapshotSecret     []byte
	snapshotSecretErr  error
)

// Claims 自定义Token结构
type Claims struct {
	UserID uint // 存用户ID
	jwt.RegisteredClaims
}

// GenerateToken 生成Token
func GenerateToken(userID uint) (string, error) {
	secret, err := getJWTSecret()
	if err != nil {
		return "", err
	}

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
	return token.SignedString(secret)
}

// ParseToken 解析Token
func ParseToken(tokenString string) (*Claims, error) {
	secret, err := getJWTSecret()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func getJWTSecret() ([]byte, error) {
	jwtSecretOnce.Do(func() {
		secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
		if len(secret) < 32 {
			jwtSecretErr = errors.New("JWT_SECRET must be set and at least 32 characters")
			return
		}
		jwtSecret = []byte(secret)
	})
	return jwtSecret, jwtSecretErr
}

func GetSnapshotSecret() ([]byte, error) {
	snapshotSecretOnce.Do(func() {
		secret := strings.TrimSpace(os.Getenv("SNAPSHOT_SECRET"))
		if len(secret) < 32 {
			secret = strings.TrimSpace(os.Getenv("JWT_SECRET"))
		}
		if len(secret) < 32 {
			snapshotSecretErr = errors.New("SNAPSHOT_SECRET or JWT_SECRET must be set and at least 32 characters")
			return
		}
		snapshotSecret = []byte(secret)
	})
	return snapshotSecret, snapshotSecretErr
}
