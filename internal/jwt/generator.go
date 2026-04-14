package jwt

import (
	"context"
	"crypto/sha256"
	"encoding/base64"

	"github.com/go-oauth2/oauth2/v4"
	"github.com/golang-jwt/jwt/v5"
)

// RolesForUserFunc загружает роли субъекта по user_id из OAuth (пустой id — client_credentials).
type RolesForUserFunc func(ctx context.Context, userID string) ([]string, error)

// AccessGenerate генерирует JWT access token для OAuth2.
type AccessGenerate struct {
	SignedKey    []byte
	SignedMethod jwt.SigningMethod
	RolesForUser RolesForUserFunc
}

// NewAccessGenerate создает генератор JWT access token.
func NewAccessGenerate(key []byte, method jwt.SigningMethod, rolesForUser RolesForUserFunc) *AccessGenerate {
	return &AccessGenerate{
		SignedKey:    key,
		SignedMethod: method,
		RolesForUser: rolesForUser,
	}
}

// Token generates JWT access token
func (a *AccessGenerate) Token(ctx context.Context, data *oauth2.GenerateBasic, isGenRefresh bool) (access, refresh string, err error) {
	var roles []string
	if a.RolesForUser != nil {
		roles, err = a.RolesForUser(ctx, data.UserID)
		if err != nil {
			return "", "", err
		}
	}

	claims := jwt.MapClaims{
		"aud": data.Client.GetID(),
		"sub": data.UserID,
		"exp": data.TokenInfo.GetAccessCreateAt().Add(data.TokenInfo.GetAccessExpiresIn()).Unix(),
		"iat": data.TokenInfo.GetAccessCreateAt().Unix(),
	}
	if roles == nil {
		roles = []string{}
	}
	claims["roles"] = roles

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
