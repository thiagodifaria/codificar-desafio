param(
    [ValidateSet("up", "down", "logs", "seed", "reset", "ps")]
    [string]$Action = "up"
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$Compose = Join-Path $Root "infra\compose.yaml"
$DatabaseUser = if ($env:POSTGRES_USER) { $env:POSTGRES_USER } else { "chamados" }
$DatabaseName = if ($env:POSTGRES_DB) { $env:POSTGRES_DB } else { "chamados" }
$WebPort = if ($env:WEB_PORT) { $env:WEB_PORT } else { "3000" }

function Assert-LastExit {
    if ($LASTEXITCODE -ne 0) {
        throw "O comando anterior falhou com código $LASTEXITCODE."
    }
}

function Invoke-Seed {
    docker compose -f $Compose exec -T database `
        psql -v ON_ERROR_STOP=1 `
        -U $DatabaseUser `
        -d $DatabaseName `
        -f /database/seed.sql
    Assert-LastExit
}

switch ($Action) {
    "up" {
        docker compose -f $Compose up -d --build
        Assert-LastExit
        Invoke-Seed
        Write-Host ""
        Write-Host "Aplicação disponível em http://localhost:$WebPort" -ForegroundColor Green
    }
    "down" {
        docker compose -f $Compose down
        Assert-LastExit
    }
    "logs" {
        docker compose -f $Compose logs -f --tail=150
        Assert-LastExit
    }
    "seed" {
        Invoke-Seed
    }
    "reset" {
        docker compose -f $Compose down -v
        Assert-LastExit
        docker compose -f $Compose up -d --build
        Assert-LastExit
        Invoke-Seed
    }
    "ps" {
        docker compose -f $Compose ps
        Assert-LastExit
    }
}
