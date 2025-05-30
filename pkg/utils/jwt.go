package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Utilidades para manejo de JWT

// Claims representa la estructura de datos del token JWT
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateJWT genera un nuevo token JWT
func GenerateJWT(userID string, secret string, expiry string) (string, error) {
	// Parsear la duración del token
	duration, err := time.ParseDuration(expiry)
	if err != nil {
		return "", err
	}

	// Crear los claims
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Crear el token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Firmar el token
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateJWT valida un token JWT y retorna los claims
func ValidateJWT(tokenString string, secret string) (*Claims, error) {
	// Parsear el token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verificar el método de firma
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("método de firma inválido")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	// Verificar que el token sea válido
	if !token.Valid {
		return nil, errors.New("token inválido")
	}

	// Extraer los claims
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("claims inválidos")
	}

	return claims, nil
}
