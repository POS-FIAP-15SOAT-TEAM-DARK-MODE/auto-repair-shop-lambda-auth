// Package authtoken mirrors the claim shape and signing method of
// auto-repair-shop's internal/pkg/auth package (UserClaims: user_id + roles,
// HS256, JWT_SECRET) so tokens issued here are accepted by the app's existing
// auth middleware without any changes on that side.
package authtoken

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserClaims struct {
	UserId string   `json:"user_id"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

func GenerateToken(secret []byte, userId string, roles []string, expiresAt time.Time) (string, error) {
	claims := UserClaims{
		UserId: userId,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}
