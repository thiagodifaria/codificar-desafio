$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$Compose = Join-Path $Root "infra\compose.yaml"

function Assert-LastExit {
    if ($LASTEXITCODE -ne 0) {
        throw "O comando anterior falhou com código $LASTEXITCODE."
    }
}

Write-Host "==> Banco isolado para testes de integração" -ForegroundColor Cyan
docker compose -f $Compose up -d --wait --wait-timeout 60 database
Assert-LastExit
docker compose -f $Compose exec -T database `
    psql -v ON_ERROR_STOP=1 -U chamados -d postgres `
    -c "DROP DATABASE IF EXISTS chamados_test WITH (FORCE);"
Assert-LastExit
docker compose -f $Compose exec -T database `
    psql -v ON_ERROR_STOP=1 -U chamados -d postgres `
    -c "CREATE DATABASE chamados_test;"
Assert-LastExit
docker compose -f $Compose exec -T database `
    sh -c 'for file in /database/migrations/*.sql; do psql -v ON_ERROR_STOP=1 -U chamados -d chamados_test -f "$file"; done'
Assert-LastExit

Write-Host "==> Testes unitários, integração e análise do backend" -ForegroundColor Cyan
docker run --rm `
    --network codificar-chamados_default `
    -e TEST_DATABASE_URL="postgres://chamados:chamados@database:5432/chamados_test?sslmode=disable" `
    -v "${Root}\service-api\service-golang:/src" `
    -w /src `
    golang:1.26-alpine `
    sh -c "go test ./... && go vet ./..."
Assert-LastExit

Write-Host "==> Contrato, typecheck e build do frontend" -ForegroundColor Cyan
docker build `
    -f (Join-Path $Root "client-web\Dockerfile") `
    -t codificar-chamados-web:test `
    $Root
Assert-LastExit

Write-Host "==> Configuração do Docker Compose" -ForegroundColor Cyan
docker compose -f $Compose config --quiet
Assert-LastExit

Write-Host "Validação concluída." -ForegroundColor Green
