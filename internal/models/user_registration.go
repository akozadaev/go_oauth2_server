package models

// UserRegistrationRequest represents user registration request
// @Description Данные для регистрации нового пользователя
type UserRegistrationRequest struct {
	// Имя пользователя (логин)
	// @example john_doe
	// @minLength 3
	// @maxLength 50
	Username string `json:"username" binding:"required,min=3,max=50"`

	// Пароль пользователя
	// @example securepassword123
	// @minLength 8
	// @maxLength 100
	Password string `json:"password" binding:"required,min=8,max=100"`

	// Email пользователя (опционально)
	// @example john@example.com
	// @format email
	Email string `json:"email,omitempty" binding:"omitempty,email"`
}
