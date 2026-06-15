#!/usr/bin/env sh
set -eu

ROOT="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
COMPOSE="$ROOT/infra/compose.yaml"
ACTION="${1:-up}"

seed() {
  docker compose -f "$COMPOSE" exec -T database \
    psql -v ON_ERROR_STOP=1 \
    -U "${POSTGRES_USER:-chamados}" \
    -d "${POSTGRES_DB:-chamados}" \
    -f /database/seed.sql
}

case "$ACTION" in
  up)
    docker compose -f "$COMPOSE" up -d --build
    seed
    printf '\nAplicação disponível em http://localhost:%s\n' "${WEB_PORT:-3000}"
    ;;
  down)
    docker compose -f "$COMPOSE" down
    ;;
  logs)
    docker compose -f "$COMPOSE" logs -f --tail=150
    ;;
  seed)
    seed
    ;;
  reset)
    docker compose -f "$COMPOSE" down -v
    docker compose -f "$COMPOSE" up -d --build
    seed
    ;;
  ps)
    docker compose -f "$COMPOSE" ps
    ;;
  *)
    echo "Uso: ./scripts/build.sh [up|down|logs|seed|reset|ps]" >&2
    exit 1
    ;;
esac

