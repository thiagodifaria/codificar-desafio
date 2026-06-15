#!/usr/bin/env sh
set -eu

ROOT="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
COMPOSE="$ROOT/infra/compose.yaml"
DATABASE_USER="${POSTGRES_USER:-chamados}"
DATABASE_PASSWORD="${POSTGRES_PASSWORD:-chamados}"
TEST_DATABASE="${POSTGRES_TEST_DB:-chamados_test}"

echo "==> Banco isolado para testes de integração"
docker compose -f "$COMPOSE" up -d --wait --wait-timeout 60 database
docker compose -f "$COMPOSE" exec -T database \
  psql -v ON_ERROR_STOP=1 -U "$DATABASE_USER" -d postgres \
  -c "DROP DATABASE IF EXISTS $TEST_DATABASE WITH (FORCE);"
docker compose -f "$COMPOSE" exec -T database \
  psql -v ON_ERROR_STOP=1 -U "$DATABASE_USER" -d postgres \
  -c "CREATE DATABASE $TEST_DATABASE;"
MSYS2_ARG_CONV_EXCL="/database/migrate.sh" docker compose -f "$COMPOSE" exec -T database \
  env PGUSER="$DATABASE_USER" PGDATABASE="$TEST_DATABASE" PGPASSWORD="$DATABASE_PASSWORD" \
  sh /database/migrate.sh

echo "==> Testes unitários, integração e análise do backend"
MSYS_NO_PATHCONV=1 docker run --rm \
  --network codificar-chamados_default \
  -e TEST_DATABASE_URL="postgres://$DATABASE_USER:$DATABASE_PASSWORD@database:5432/$TEST_DATABASE?sslmode=disable" \
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
