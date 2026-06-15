$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$Compose = Join-Path $Root "infra\compose.yaml"
$DatabaseUser = if ($env:POSTGRES_USER) { $env:POSTGRES_USER } else { "chamados" }
$DatabasePassword = if ($env:POSTGRES_PASSWORD) { $env:POSTGRES_PASSWORD } else { "chamados" }
$TestDatabase = if ($env:POSTGRES_TEST_DB) { $env:POSTGRES_TEST_DB } else { "chamados_test" }

function Assert-LastExit {
    if ($LASTEXITCODE -ne 0) {
        throw "O comando anterior falhou com código $LASTEXITCODE."
    }
}

Write-Host "==> Banco isolado para testes de integração" -ForegroundColor Cyan
docker compose -f $Compose up -d --wait --wait-timeout 60 database
Assert-LastExit
docker compose -f $Compose exec -T database `
    psql -v ON_ERROR_STOP=1 -U $DatabaseUser -d postgres `
    -c "DROP DATABASE IF EXISTS $TestDatabase WITH (FORCE);"
Assert-LastExit
docker compose -f $Compose exec -T database `
    psql -v ON_ERROR_STOP=1 -U $DatabaseUser -d postgres `
    -c "CREATE DATABASE $TestDatabase;"
Assert-LastExit
docker compose -f $Compose exec -T database `
    env PGUSER=$DatabaseUser PGDATABASE=$TestDatabase PGPASSWORD=$DatabasePassword `
    sh /database/migrate.sh
Assert-LastExit

Write-Host "==> Testes unitários, integração e análise do backend" -ForegroundColor Cyan
docker run --rm `
    --network codificar-chamados_default `
    -e TEST_DATABASE_URL="postgres://${DatabaseUser}:${DatabasePassword}@database:5432/${TestDatabase}?sslmode=disable" `
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
