#!/bin/bash
# curl-test.sh — тестирование OAuth2 API через curl
# Использование: ./scripts/curl-test.sh [BASE_URL]
# Пример: ./scripts/curl-test.sh http://localhost:8080

set -e

BASE_URL="${1:-http://localhost:8080}"
BOLD="\033[1m"
GREEN="\033[32m"
BLUE="\033[34m"
YELLOW="\033[33m"
NC="\033[0m"

echo -e "${BOLD}═══ OAuth2 Server API Test ═══${NC}"
echo -e "Base URL: ${BLUE}${BASE_URL}${NC}"
echo ""

# Проверка доступности
echo -e "${BOLD}1. Health Check${NC}"
echo "GET $BASE_URL/health"
curl -s -w "\nHTTP %{http_code}\n" "$BASE_URL/health" | tail -20
echo ""

# Уникальные данные для каждого запуска (избегаем конфликтов при повторных запусках)
TEST_USER="testuser_$$"
TEST_PASS="testpass123"

# Регистрация пользователя
echo -e "${BOLD}2. Регистрация пользователя (POST /users)${NC}"
echo "POST $BASE_URL/users"
USER_RESPONSE=$(curl -s -X POST "$BASE_URL/users" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${TEST_USER}\",\"password\":\"${TEST_PASS}\"}")
echo "$USER_RESPONSE"

if command -v jq &>/dev/null; then
  USER_ID=$(echo "$USER_RESPONSE" | jq -r '.user_id // empty')
else
  USER_ID=$(echo "$USER_RESPONSE" | grep -o '"user_id":"[^"]*"' | cut -d'"' -f4)
fi
echo ""
echo ""

# Регистрация клиента (с user_id от созданного пользователя)
echo -e "${BOLD}3. Регистрация клиента (POST /clients)${NC}"
echo "POST $BASE_URL/clients"
CLIENT_PAYLOAD="{\"domain\":\"http://localhost:3000\",\"user_id\":\"${USER_ID}\"}"
if [ -z "$USER_ID" ]; then
  CLIENT_PAYLOAD="{\"domain\":\"http://localhost:3000\",\"username\":\"${TEST_USER}\",\"password\":\"${TEST_PASS}\"}"
fi
CLIENT_RESPONSE=$(curl -s -X POST "$BASE_URL/clients" \
  -H "Content-Type: application/json" \
  -d "$CLIENT_PAYLOAD")
echo "$CLIENT_RESPONSE"

if command -v jq &>/dev/null; then
  CLIENT_ID=$(echo "$CLIENT_RESPONSE" | jq -r '.client_id // empty')
  CLIENT_SECRET=$(echo "$CLIENT_RESPONSE" | jq -r '.client_secret // empty')
else
  CLIENT_ID=$(echo "$CLIENT_RESPONSE" | grep -o '"client_id":"[^"]*"' | cut -d'"' -f4)
  CLIENT_SECRET=$(echo "$CLIENT_RESPONSE" | grep -o '"client_secret":"[^"]*"' | cut -d'"' -f4)
fi

if [ -z "$CLIENT_ID" ] || [ -z "$CLIENT_SECRET" ]; then
  echo -e "${YELLOW}⚠ Не удалось создать клиента. Проверьте, что сервер запущен и пользователь создан.${NC}"
  echo "Запуск прерван."
  exit 1
fi
echo ""

# Получение токена (password grant)
echo -e "${BOLD}4. Получение токена — Password Grant (POST /token)${NC}"
echo "POST $BASE_URL/token (grant_type=password)"
TOKEN_RESPONSE=$(curl -s -X POST "$BASE_URL/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=password&client_id=${CLIENT_ID}&client_secret=${CLIENT_SECRET}&username=${TEST_USER}&password=${TEST_PASS}")
echo "$TOKEN_RESPONSE"

if command -v jq &>/dev/null; then
  ACCESS_TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.access_token // empty')
  REFRESH_TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.refresh_token // empty')
else
  ACCESS_TOKEN=$(echo "$TOKEN_RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
  REFRESH_TOKEN=$(echo "$TOKEN_RESPONSE" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)
fi
echo ""

if [ -n "$ACCESS_TOKEN" ] && [ "$ACCESS_TOKEN" != "null" ]; then
  # Интроспекция токена
  echo -e "${BOLD}5. Интроспекция токена (POST /introspect)${NC}"
  echo "POST $BASE_URL/introspect"
  curl -s -X POST "$BASE_URL/introspect" \
    -H "Content-Type: application/json" \
    -d "{\"token\":\"${ACCESS_TOKEN}\"}" | head -20
  echo ""
  echo ""

  # Обновление токена (опционально)
  if [ -n "$REFRESH_TOKEN" ] && [ "$REFRESH_TOKEN" != "null" ]; then
    echo -e "${BOLD}6. Обновление токена — Refresh Grant (POST /token)${NC}"
    echo "POST $BASE_URL/token (grant_type=refresh_token)"
    curl -s -X POST "$BASE_URL/token" \
      -H "Content-Type: application/x-www-form-urlencoded" \
      -d "grant_type=refresh_token&client_id=${CLIENT_ID}&client_secret=${CLIENT_SECRET}&refresh_token=${REFRESH_TOKEN}" | head -20
    echo ""
  fi
fi

echo ""
echo -e "${BOLD}═══ Готовые curl-команды для ручного тестирования ═══${NC}"
echo ""
echo "# Health:"
echo "curl -s $BASE_URL/health | jq ."
echo ""
echo "# Регистрация пользователя:"
echo "curl -X POST $BASE_URL/users -H 'Content-Type: application/json' -d '{\"username\":\"myuser\",\"password\":\"mypass\"}'"
echo ""
echo "# Регистрация клиента (с созданием пользователя):"
echo "curl -X POST $BASE_URL/clients -H 'Content-Type: application/json' -d '{\"domain\":\"http://localhost:3000\",\"username\":\"myuser\",\"password\":\"mypass\"}'"
echo ""
echo "# Token (password grant) — подставьте client_id и client_secret:"
echo "curl -X POST $BASE_URL/token -H 'Content-Type: application/x-www-form-urlencoded' -d 'grant_type=password&client_id=CLIENT_ID&client_secret=CLIENT_SECRET&username=myuser&password=mypass'"
echo ""
echo "# Introspect:"
echo "curl -X POST $BASE_URL/introspect -H 'Content-Type: application/json' -d '{\"token\":\"YOUR_ACCESS_TOKEN\"}'"
echo ""
echo "# Authorize (POST с username/password):"
echo "curl -X POST $BASE_URL/authorize -H 'Content-Type: application/json' -d '{\"response_type\":\"code\",\"client_id\":\"CLIENT_ID\",\"redirect_uri\":\"http://localhost:3000/callback\",\"username\":\"myuser\",\"password\":\"mypass\"}'"
echo ""
echo "# Metrics (Prometheus):"
echo "curl -s $BASE_URL/metrics | head -30"
