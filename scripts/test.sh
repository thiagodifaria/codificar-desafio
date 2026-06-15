#!/usr/bin/env sh
set -eu

ROOT="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
COMPOSE="$ROOT/infra/compose.yaml"

echo "==> Banco isolado para testes de integração"
docker compose -f "$COMPOSE" up -d --wait --wait-timeout 60 database
docker compose -f "$COMPOSE" exec -T database \
  psql -v ON_ERROR_STOP=1 -U chamados -d postgres \
  -c "DROP DATABASE IF EXISTS chamados_test WITH (FORCE);"
docker compose -f "$COMPOSE" exec -T database \
  psql -v ON_ERROR_STOP=1 -U chamados -d postgres \
  -c "CREATE DATABASE chamados_test;"
docker compose -f "$COMPOSE" exec -T database \
  sh -c 'for file in /database/migrations/*.sql; do psql -v ON_ERROR_STOP=1 -U chamados -d chamados_test -f "$file"; done'

echo "==> Testes unitários, integração e análise do backend"
MSYS_NO_PATHCONV=1 docker run --rm \
  --network codificar-chamados_default \
  -e TEST_DATABASE_URL="postgres://chamados:chamados@database:5432/chamados_test?sslmode=disable" \
  -v "$ROOT/service-api/service-golang:/src" \
  -w /src \
  golang:1.26-alpine \
  sh -c "go test ./... && go vet ./..."

echo "==> Contrato, typecheck e build do frontend"
docker build \
  -f "$ROOT/client-web/Dockerfile" \
  -t codificar-chamados-web:test \
  "$ROOT"

echo "==> Configuração do Docker Compose"
docker compose -f "$COMPOSE" config --quiet

echo "Validação concluída."
