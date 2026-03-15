package utils

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

// Claims holds JWT claims (subject = user ID).
type Claims struct {
	jwt.RegisteredClaims
	UserID int `json:"user_id"`
}

// NewToken creates a JWT for the given user ID, valid for 7 days.
func NewToken(userID int, secret string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.Itoa(userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

// ParseToken parses a JWT and returns the user ID. Returns ErrInvalidToken on failure.
func ParseToken(tokenString, secret string) (userID int, err error) {
	t, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return 0, ErrInvalidToken
	}
	claims, ok := t.Claims.(*Claims)
	if !ok || !t.Valid {
		return 0, ErrInvalidToken
	}
	return claims.UserID, nil
}
