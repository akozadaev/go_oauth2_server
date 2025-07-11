package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"go_oauth2_server/internal/config"
	"go_oauth2_server/internal/jwt"
	"go_oauth2_server/internal/models"
	"go_oauth2_server/internal/storage"

	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/manage"
	"github.com/go-oauth2/oauth2/v4/server"
	jwtLib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Handler struct {
	store  *storage.PostgresStore
	logger *slog.Logger
	srv    *server.Server
	config *config.Config
}

func New(store *storage.PostgresStore, logger *slog.Logger, cfg *config.Config) *Handler {
	manager := manage.NewDefaultManager()

	// Конфигурация токенов
	manager.SetAuthorizeCodeTokenCfg(manage.DefaultAuthorizeCodeTokenCfg)
	manager.SetRefreshTokenCfg(manage.DefaultRefreshTokenCfg)

	// Генерация JWT access токенов
	jwtGen := jwt.NewJWTAccessGenerate([]byte(cfg.JWTSecret), jwtLib.SigningMethodHS256)
	manager.MapAccessGenerate(jwtGen)

	// Хранилище клиентов
	manager.MapClientStorage(store.GetClientStore())

	// Хранилище токенов
	manager.MapTokenStorage(store.GetTokenStore())

	srv := server.NewDefaultServer(manager)
	srv.SetAllowGetAccessRequest(true)
	srv.SetClientInfoHandler(server.ClientFormHandler)

	// Обработка авторизации по логину и паролю
	srv.SetPasswordAuthorizationHandler(func(ctx context.Context, clientID, username, password string) (userID string, err error) {
		user, err := store.ValidateUser(ctx, username, password)
		if err != nil {
			return "", err
		}
		return user.ID, nil
	})

	// Обработка пользовательской авторизации
	srv.SetUserAuthorizationHandler(func(w http.ResponseWriter, r *http.Request) (userID string, err error) {
		return r.FormValue("user_id"), nil
	})

	// Обработка авторизации клиента
	srv.SetClientAuthorizedHandler(func(clientID string, grant oauth2.GrantType) (allowed bool, err error) {
		// Разрешаем все grant типы для простоты — в проде стоит сделать полноценную проверку
		return true, nil
	})

	return &Handler{
		store:  store,
		logger: logger,
		srv:    srv,
		config: cfg,
	}
}

// AuthorizeGet godoc
// @Summary Авторизация (GET)
// @Description Авторизация пользователя (через браузер)
// @Tags authorize
// @Accept json
// @Produce html
// @Success 200 {string} string "HTML-форма"
// @Failure 400 {object} map[string]string "Неверный запрос. Пример:
// {
//   \"error\": \"invalid_request\",
//   \"error_description\": \"Missing parameters\"
// }"
// @Failure 401 {object} map[string]string "Ошибка авторизации:
//{
//\"error\": \"access_denied\",
//\"error_description\": \"Invalid credentials\"
//}"
// @Router /authorize [get]

// AuthorizePost godoc
// @Summary Авторизация (POST)
// @Description Авторизация пользователя с передачей формы
// @Tags authorize
// @Accept json
// @Produce html
// @Failure 400 {object} map[string]string "Неверный запрос. Пример:
//
//	{
//	  \"error\": \"invalid_request\",
//	  \"error_description\": \"Missing parameters\"
//	}"
//
// @Failure 401 {object} map[string]string "Ошибка авторизации:
// {
// \"error\": \"access_denied\",
// \"error_description\": \"Invalid credentials\"
// }"
// @Router /authorize [post]
func (h *Handler) Authorize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method == "POST" {
		var req models.AuthorizeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.logger.Error("Failed to decode authorize request", "error", err)
			h.writeErrorResponse(w, "invalid_request", "Invalid request format", http.StatusBadRequest)
			return
		}

		// Проверка обязательных полей
		if req.ClientID == "" || req.ResponseType == "" {
			h.writeErrorResponse(w, "invalid_request", "Missing required parameters", http.StatusBadRequest)
			return
		}

		// Проверка логина и пароля, если они переданы
		if req.Username != "" && req.Password != "" {
			user, err := h.store.ValidateUser(ctx, req.Username, req.Password)
			if err != nil {
				h.logger.Error("Invalid user credentials", "username", req.Username, "error", err)
				h.writeErrorResponse(w, "access_denied", "Invalid credentials", http.StatusUnauthorized)
				return
			}

			// Установка параметров формы для сервера OAuth2
			if r.Form == nil {
				r.Form = url.Values{}
			}
			r.Form.Set("response_type", req.ResponseType)
			r.Form.Set("client_id", req.ClientID)
			r.Form.Set("redirect_uri", req.RedirectURI)
			r.Form.Set("scope", req.Scope)
			r.Form.Set("state", req.State)
			r.Form.Set("user_id", user.ID)
		}
	}

	if err := h.srv.HandleAuthorizeRequest(w, r); err != nil {
		h.logger.Error("Authorization request failed", "error", err)
		h.writeErrorResponse(w, "server_error", "Authorization failed", http.StatusInternalServerError)
	}
}

// Token godoc
// @Summary Обмен кода на токен
// @Description АОбмен кода на токен
// @Tags token
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
//
//	  \"error\": \"invalid_request\",
//	  \"error_description\": \"Missing parameters\"
//	}" "Успешный ответ с полной информацией о добавленных ссылках"
//
// @Failure 400 {object} map[string]string "Неверный запрос. Пример:
//
//	{
//	  \"error\": \"invalid_request\",
//	  \"error_description\": \"Missing parameters\"
//	}"
//
// @Failure 401 {object} map[string]string "Ошибка авторизации:
// {
// \"error\": \"access_denied\",
// \"error_description\": \"Invalid credentials\"
// }"
// @Router /token [post]
func (h *Handler) Token(w http.ResponseWriter, r *http.Request) {
	if err := h.srv.HandleTokenRequest(w, r); err != nil {
		h.logger.Error("Token request failed", "error", err)
		// Сервер OAuth2 сам отправит корректный ответ об ошибке
	}
}

func (h *Handler) Introspect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req models.IntrospectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode introspect request", "error", err)
		h.writeErrorResponse(w, "invalid_request", "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		h.writeErrorResponse(w, "invalid_request", "Token parameter is required", http.StatusBadRequest)
		return
	}

	// Для JWT токенов можем валидировать их напрямую
	if h.isJWTToken(req.Token) {
		response := h.validateJWTToken(req.Token)
		h.writeJSONResponse(w, response, http.StatusOK)
		return
	}

	// В противном случае к стандартной валидации через OAuth2 manager
	ti, err := h.srv.Manager.LoadAccessToken(ctx, req.Token)
	if err != nil {
		// Токен недействителен или просрочен
		response := models.IntrospectResponse{Active: false}
		h.writeJSONResponse(w, response, http.StatusOK)
		return
	}

	// Проверка срока действия токена
	expiresAt := ti.GetAccessCreateAt().Add(ti.GetAccessExpiresIn())
	if expiresAt.Before(time.Now()) {
		response := models.IntrospectResponse{Active: false}
		h.writeJSONResponse(w, response, http.StatusOK)
		return
	}

	// Токен действителен
	response := models.IntrospectResponse{
		Active:   true,
		ClientID: ti.GetClientID(),
		UserID:   ti.GetUserID(),
		Scope:    ti.GetScope(),
		Exp:      expiresAt.Unix(),
	}

	h.writeJSONResponse(w, response, http.StatusOK)
}

// isJWTToken предварительная валидация JWT,  проверяет, является ли строка JWT-токеном
func (h *Handler) isJWTToken(tokenString string) bool {
	// JWT токены состоят из трех частей, разделенных точками
	parts := len(tokenString) > 0 && len(tokenString) < 2048
	return parts && (tokenString[0] == 'e' || tokenString[0] == 'E') // JWT обычно начинается с eyJ
}

// validateJWTToken прямая валидация JWT-токена
func (h *Handler) validateJWTToken(tokenString string) models.IntrospectResponse {
	token, err := jwtLib.Parse(tokenString, func(token *jwtLib.Token) (interface{}, error) {
		// Проверка метода подписи
		if _, ok := token.Method.(*jwtLib.SigningMethodHMAC); !ok {
			return nil, jwt.ErrInvalidSigningMethod
		}
		return []byte(h.config.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return models.IntrospectResponse{Active: false}
	}

	claims, ok := token.Claims.(jwtLib.MapClaims)
	if !ok {
		return models.IntrospectResponse{Active: false}
	}

	// Проверка срока действия
	if exp, ok := claims["exp"].(float64); ok {
		if time.Unix(int64(exp), 0).Before(time.Now()) {
			return models.IntrospectResponse{Active: false}
		}
	}

	// Извлечение данных из claims
	clientID, _ := claims["aud"].(string)
	username, _ := claims["sub"].(string)
	exp, _ := claims["exp"].(float64)

	return models.IntrospectResponse{
		Active:   true,
		ClientID: clientID,
		UserID:   username,
		Scope:    "", // Сейчас не используем
		Exp:      int64(exp),
	}
}

func (h *Handler) RegisterClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Domain      string   `json:"domain"`
		UserID      string   `json:"user_id"`
		Username    string   `json:"username"`
		Password    string   `json:"password"`
		RedirectURI []string `json:"redirect_uris,omitempty"`
		GrantTypes  []string `json:"grant_types,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode client registration request", "error", err)
		h.writeErrorResponse(w, "invalid_request", "Invalid request format", http.StatusBadRequest)
		return
	}

	// Создаем пользователя, если указаны username и password
	if req.Username != "" && req.Password != "" {
		user := &models.User{
			ID:        uuid.New().String(),
			Username:  req.Username,
			Password:  req.Password,
			CreatedAt: time.Now(),
		}

		if err := h.store.CreateUser(ctx, user); err != nil {
			h.logger.Error("Failed to create user", "error", err)
			h.writeErrorResponse(w, "server_error", "Failed to create user", http.StatusInternalServerError)
			return
		}

		req.UserID = user.ID
	}

	// Создаем клиента
	client := &models.Client{
		ID:        uuid.New().String(),
		Secret:    uuid.New().String(),
		Domain:    req.Domain,
		UserID:    req.UserID,
		CreatedAt: time.Now(),
	}

	if err := h.store.CreateClient(ctx, client); err != nil {
		h.logger.Error("Failed to create client", "error", err)
		h.writeErrorResponse(w, "server_error", "Failed to create client", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"client_id":     client.ID,
		"client_secret": client.Secret,
		"domain":        client.Domain,
		"user_id":       client.UserID,
		"created_at":    client.CreatedAt.Unix(),
	}

	h.logger.Info("Client registered successfully", "client_id", client.ID, "domain", client.Domain)
	h.writeJSONResponse(w, response, http.StatusCreated)
}

func (h *Handler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode user registration request", "error", err)
		h.writeErrorResponse(w, "invalid_request", "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		h.writeErrorResponse(w, "invalid_request", "Username and password are required", http.StatusBadRequest)
		return
	}

	user := &models.User{
		ID:        uuid.New().String(),
		Username:  req.Username,
		Password:  req.Password,
		CreatedAt: time.Now(),
	}

	if err := h.store.CreateUser(ctx, user); err != nil {
		h.logger.Error("Failed to create user", "error", err)
		h.writeErrorResponse(w, "server_error", "Failed to create user", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"user_id":    user.ID,
		"username":   user.Username,
		"created_at": user.CreatedAt.Unix(),
	}

	h.logger.Info("User registered successfully", "user_id", user.ID, "username", user.Username)
	h.writeJSONResponse(w, response, http.StatusCreated)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Проверка подключения к базе данных
	if err := h.store.Ping(ctx); err != nil {
		h.logger.Error("Database health check failed", "error", err)
		h.writeJSONResponse(w, map[string]string{
			"status": "unhealthy",
			"error":  "database connection failed",
		}, http.StatusServiceUnavailable)
		return
	}

	h.writeJSONResponse(w, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	}, http.StatusOK)
}

func (h *Handler) writeJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

func (h *Handler) writeErrorResponse(w http.ResponseWriter, errorCode, description string, statusCode int) {
	response := map[string]string{
		"error":             errorCode,
		"error_description": description,
	}
	h.writeJSONResponse(w, response, statusCode)
}
