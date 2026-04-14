// Package jwt содержит генерацию и валидацию JWT для OAuth2.
package jwt

import "errors"

var (
	// ErrInvalidSigningMethod возвращается при неподдерживаемом алгоритме подписи JWT.
	ErrInvalidSigningMethod = errors.New("invalid signing method")
)
