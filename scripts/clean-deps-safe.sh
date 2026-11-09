#!/bin/bash

# Проверка зависимостей; изменения и очистка включаются явно.
set -Eeuo pipefail

stage="инициализация"
trap 'status=$?; printf "Ошибка на этапе: %s (код %s)\n" "$stage" "$status" >&2; exit "$status"' ERR
fail() { printf 'Ошибка: %s\n' "$*" >&2; exit 1; }
usage() {
    cat <<'HELP'
Использование: clean-deps-safe.sh [--tidy] [--clean-cache] [--remove-vendor]
Без параметров: загрузить и проверить зависимости.
  --tidy           Актуализировать go.mod и go.sum
  --clean-cache    Очистить используемый кеш модулей перед загрузкой
  --remove-vendor  Удалить vendor в корне проекта
  --help           Показать справку
Параметры можно комбинировать. GOMODCACHE берётся из окружения Go.
HELP
}

tidy=false
clean_cache=false
remove_vendor=false
for arg in "$@"; do
    case "$arg" in
        --tidy) tidy=true ;;
        --clean-cache) clean_cache=true ;;
        --remove-vendor) remove_vendor=true ;;
        --help|-h) usage; exit 0 ;;
        *) usage >&2; fail "Неизвестный параметр: $arg" ;;
    esac
done

command -v go >/dev/null 2>&1 || fail "Команда go не найдена в PATH"
project_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)"
cd -- "$project_root"
[[ -f go.mod ]] || fail "Не найден $project_root/go.mod"
# Запись нужна также go mod download, если требуется дополнить go.sum.
[[ -w . ]] || fail "Нет прав записи в $project_root"
for manifest in go.mod go.sum; do
    if [[ -e "$manifest" && ! -w "$manifest" ]]; then
        fail "Нет прав записи в $manifest"
    fi
done

stage="проверка кеша модулей"
modcache="$(go env GOMODCACHE)"
[[ -n "$modcache" && "$modcache" == /* && "$modcache" != / ]] || fail "Некорректный GOMODCACHE: $modcache"
mkdir -p -- "$modcache"
probe="$(mktemp "$modcache/.write-check.XXXXXXXX")"
rm -- "$probe"

if "$clean_cache"; then
    stage="очистка кеша модулей"
    printf 'Очистка кеша модулей: %s\n' "$modcache"
    go clean -modcache
fi
if "$remove_vendor"; then
    stage="удаление vendor"
    rm -rf -- "$project_root/vendor"
fi
if "$tidy"; then
    stage="актуализация go.mod и go.sum"
    go mod tidy
fi
stage="загрузка модулей"
go mod download
stage="проверка модулей"
go mod verify
printf 'Зависимости загружены и проверены.\n'
