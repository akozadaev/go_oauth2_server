package jwt

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"github.com/go-oauth2/oauth2/v4"
	"github.com/golang-jwt/jwt/v5"
)

// JWTAccessGenerate JWT access token generator
type JWTAccessGenerate struct {
	SignedKey    []byte
	SignedMethod jwt.SigningMethod
}

// NewJWTAccessGenerate создает экземпляр токена доступа jwt
func NewJWTAccessGenerate(key []byte, method jwt.SigningMethod) *JWTAccessGenerate {
	return &JWTAccessGenerate{
		SignedKey:    key,
		SignedMethod: method,
	}
}

// Token generates JWT access token
func (a *JWTAccessGenerate) Token(ctx context.Context, data *oauth2.GenerateBasic, isGenRefresh bool) (access, refresh string, err error) {

	claims := jwt.MapClaims{
		"aud": data.Client.GetID(),
		"sub": data.UserID,
		"exp": data.TokenInfo.GetAccessCreateAt().Add(data.TokenInfo.GetAccessExpiresIn()).Unix(),
		"iat": data.TokenInfo.GetAccessCreateAt().Unix(),
	}

	token := jwt.NewWithClaims(a.SignedMethod, claims)
	access, err = token.SignedString(a.SignedKey)
	if err != nil {
		return "", "", err
	}

	refresh = ""
	if isGenRefresh {
		t := sha256.Sum256([]byte(access))
		refresh = base64.URLEncoding.EncodeToString(t[:])
	}

	return access, refresh, nil
}
