package models

// ClientRegistrationRequest represents client registration request
// @Description Данные для регистрации клиента OAuth2
type ClientRegistrationRequest struct {
	// Домен клиента
	// @example example.com
	Domain string `json:"domain" binding:"required"`

	// ID существующего пользователя (опционально)
	// @example user123
	UserID string `json:"user_id,omitempty"`

	// Имя пользователя для создания нового пользователя (опционально)
	// @example admin
	Username string `json:"username,omitempty"`

	// Пароль для создания нового пользователя (опционально)
	// @example securepassword123
	Password string `json:"password,omitempty"`

	// Список URI для перенаправления
	// @example ["https://example.com/callback"]
	RedirectURI []string `json:"redirect_uris,omitempty"`

	// Список разрешенных типов грантов
	// @example ["authorization_code","client_credentials"]
	GrantTypes []string `json:"grant_types,omitempty"`
}
