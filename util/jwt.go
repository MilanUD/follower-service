package util

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/metadata"
)

var jwtKey = func() []byte {
	// U produkciji OBAVEZNO setovati kroz env var:
	// u docker-compose: JWT_SECRET=neka_jaka_lozinka
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// dev fallback – nemoj ostaviti u produkciji
		secret = "tajna_lozinka"
	}
	return []byte(secret)
}()

type Claims struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	// ostajemo kompatibilni sa Stakeholders servisom:
	Role string `json:"http://schemas.microsoft.com/ws/2008/06/identity/claims/role"`
	jwt.RegisteredClaims
}

// (Opcionalno) Generisanje za lokalni test; u realnosti token izdaje gateway/auth.
func GenerateToken(id, username, role string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	now := time.Now()
	claims := &Claims{
		ID:       id,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

// ValidateToken parsira i validira JWT i vraća claim-ove.
func ValidateToken(tokenStr string) (*Claims, error) {
	if tokenStr == "" {
		return nil, errors.New("empty token")
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		// ograniči na HMAC
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %T", t.Method)
		}
		return jwtKey, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// ExtractBearer iz Authorization metadata (gRPC): "Bearer <token>".
func ExtractBearer(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nilTokenErr()
	}
	// header keys su lowercased; "authorization" je standard
	values := md.Get("authorization")
	if len(values) == 0 {
		return nilTokenErr()
	}
	parts := strings.SplitN(values[0], " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("authorization header must be in format: Bearer <token>")
	}
	return parts[1], nil
}

func nilTokenErr() (string, error) {
	return "", errors.New("authorization metadata not found")
}
