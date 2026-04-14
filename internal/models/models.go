package models

import (
	"time"
)

// Client описывает OAuth2-клиента, зарегистрированного в системе.
type Client struct {
	ID        string    `json:"id" db:"id"`
	Secret    string    `json:"secret" db:"secret"`
	Domain    string    `json:"domain" db:"domain"`
	UserID    string    `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// User описывает учетную запись пользователя в системе.
type User struct {
	ID        string    `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	Password  string    `json:"password" db:"password"`
	Roles     []string  `json:"roles,omitempty" db:"roles"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// AuthorizeRequest описывает входные данные для OAuth2 endpoint `/authorize`.
type AuthorizeRequest struct {
	ResponseType string `json:"response_type"`
	ClientID     string `json:"client_id"`
	RedirectURI  string `json:"redirect_uri"`
	Scope        string `json:"scope"`
	State        string `json:"state"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

// TokenRequest описывает входные параметры для OAuth2 endpoint `/token`.
type TokenRequest struct {
	GrantType    string `json:"grant_type"`
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Scope        string `json:"scope"`
}

// TokenResponse описывает успешный ответ endpoint `/token`.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// IntrospectRequest описывает входные данные для endpoint `/introspect`.
type IntrospectRequest struct {
	Token         string `json:"token"`
	TokenTypeHint string `json:"token_type_hint,omitempty"`
}

// IntrospectResponse содержит результат интроспекции access token.
type IntrospectResponse struct {
	Active   bool     `json:"active"`
	ClientID string   `json:"client_id,omitempty"`
	UserID   string   `json:"user_id,omitempty"`
	Roles    []string `json:"roles,omitempty"`
	Scope    string   `json:"scope,omitempty"`
	Exp      int64    `json:"exp,omitempty"`
}
