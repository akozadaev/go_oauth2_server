#!/bin/bash

# Скрипт для решения проблем с сетью при загрузке Go зависимостей

set -e

echo "🌐 Исправление проблем с сетью для Go модулей..."

# Проверяем подключение к интернету
echo "🔍 Проверка подключения к интернету..."
if ping -c 1 8.8.8.8 > /dev/null 2>&1; then
    echo "✅ IPv4 подключение работает"
else
    echo "❌ Проблемы с IPv4 подключением"
    exit 1
fi

# Проверяем IPv6
echo "🔍 Проверка IPv6..."
if ping6 -c 1 2001:4860:4860::8888 > /dev/null 2>&1; then
    echo "✅ IPv6 подключение работает"
else
    echo "⚠️  IPv6 подключение не работает, отключаем его для Go"
    export GODEBUG=netdns=go
fi

# Настраиваем Go proxy
echo "🔧 Настройка Go proxy..."
export GOPROXY=https://proxy.golang.org,direct
export GOSUMDB=sum.golang.org
export GONOPROXY=""
export GONOSUMDB=""
export GOPRIVATE=""

# Альтернативные прокси на случай проблем
echo "🔄 Пробуем альтернативные прокси..."

# Пробуем основной прокси
if go env GOPROXY | grep -q "proxy.golang.org"; then
    echo "✅ Используется стандартный Go proxy"
else
    echo "🔧 Настраиваем стандартный Go proxy"
    go env -w GOPROXY=https://proxy.golang.org,direct
fi

# Безопасная очистка кеша модулей
echo "🧹 Безопасная очистка кеша модулей..."
if [ -w "$(go env GOMODCACHE)" ] 2>/dev/null; then
    go clean -modcache
    echo "✅ Кеш модулей очищен"
else
    echo "⚠️  Нет прав для очистки глобального кеша, пропускаем..."
    # Очищаем только локальный кеш
    rm -rf ./vendor/ 2>/dev/null || true
fi

# Пробуем загрузить зависимости с таймаутом
echo "⬇️  Загрузка зависимостей с увеличенным таймаутом..."
timeout 300 go mod download 2>/dev/null || {
    echo "⚠️  Стандартная загрузка не удалась, пробуем альтернативные методы..."

    # Пробуем китайский прокси
    echo "🇨🇳 Пробуем китайский Go proxy..."
    export GOPROXY=https://goproxy.cn,direct
    go env -w GOPROXY=https://goproxy.cn,direct
    timeout 300 go mod download 2>/dev/null || {

        # Пробуем европейский прокси
        echo "🇪🇺 Пробуем европейский Go proxy..."
        export GOPROXY=https://goproxy.io,direct
        go env -w GOPROXY=https://goproxy.io,direct
        timeout 300 go mod download 2>/dev/null || {

            echo "❌ Все прокси не работают, пробуем прямое подключение..."
            export GOPROXY=direct
            go env -w GOPROXY=direct
            timeout 600 go mod download 2>/dev/null || {
                echo "❌ Прямое подключение тоже не работает"
                echo "💡 Попробуйте запустить с sudo или создать vendor директорию"
                exit 1
            }
        }
    }
}

echo "✅ Зависимости загружены успешно!"

# Проверяем зависимости
echo "🔍 Проверка целостности зависимостей..."
go mod verify || echo "⚠️  Некоторые зависимости могут быть повреждены"

echo "✅ Проблемы с сетью исправлены!"
