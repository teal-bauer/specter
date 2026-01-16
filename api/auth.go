package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateToken creates a JWT token for Ghost Admin API authentication
// The key format is "{id}:{secret}" where id is the key ID and secret is hex-encoded
func GenerateToken(adminKey string) (string, error) {
	parts := strings.SplitN(adminKey, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid admin key format: expected 'id:secret'")
	}

	keyID := parts[0]
	secret := parts[1]

	// Decode hex secret
	secretBytes, err := hexDecode(secret)
	if err != nil {
		return "", fmt.Errorf("decoding secret: %w", err)
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
		"aud": "/admin/",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = keyID

	return token.SignedString(secretBytes)
}

func hexDecode(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, fmt.Errorf("hex string has odd length")
	}

	result := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		var b byte
		for j := 0; j < 2; j++ {
			c := s[i+j]
			var nibble byte
			switch {
			case c >= '0' && c <= '9':
				nibble = c - '0'
			case c >= 'a' && c <= 'f':
				nibble = c - 'a' + 10
			case c >= 'A' && c <= 'F':
				nibble = c - 'A' + 10
			default:
				return nil, fmt.Errorf("invalid hex character: %c", c)
			}
			b = b<<4 | nibble
		}
		result[i/2] = b
	}
	return result, nil
}
