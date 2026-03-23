package ws

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type SessionTokenClaims struct {
	DeviceID  string `json:"device_id"`
	TenantID  string `json:"tenant_id"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

func ValidateSessionToken(tokenStr, secret string) (*SessionTokenClaims, error) {
	claims := &SessionTokenClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
