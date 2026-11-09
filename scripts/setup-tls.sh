#!/bin/bash
# Локальный самоподписанный сертификат; существующие файлы не заменяются.
set -Eeuo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."
command -v openssl >/dev/null || { echo "Требуется openssl" >&2; exit 1; }
mkdir -p nginx/ssl
if [[ -e nginx/ssl/nginx.key || -e nginx/ssl/nginx.crt ]]; then
    [[ -s nginx/ssl/nginx.key && -s nginx/ssl/nginx.crt ]] || {
        echo "Нужна полная пара nginx/ssl/nginx.crt и nginx.key" >&2; exit 1;
    }
    echo "Используется существующая пара сертификата и ключа."
    exit 0
fi
umask 077
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout nginx/ssl/nginx.key -out nginx/ssl/nginx.crt \
    -subj "/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
echo "Создан локальный самоподписанный сертификат для localhost."
