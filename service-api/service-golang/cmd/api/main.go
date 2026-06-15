package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thiagodifaria/codificar-chamados/internal/config"
	"github.com/thiagodifaria/codificar-chamados/internal/database"
	"github.com/thiagodifaria/codificar-chamados/internal/httpapi"
	"github.com/thiagodifaria/codificar-chamados/internal/ticket"
)

func main() {
	// Inicializa o logger estruturado em JSON para facilitar a leitura dos eventos da API.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Carrega e valida as configurações necessárias para iniciar a aplicação.
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	// Cria um contexto cancelável pelos sinais de encerramento do sistema operacional.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Inicializa o pool de conexões com o PostgreSQL.
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Configura o servidor HTTP com limites adequados para evitar conexões indefinidas.
	ticketService := ticket.NewService(ticket.NewStore(pool))
	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           httpapi.New(ticketService, pool, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Inicia o servidor em uma goroutine para manter a rotina principal aguardando o shutdown.
	go func() {
		logger.Info("API started", "address", cfg.Address)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("API stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	// Aguarda um sinal de interrupção antes de iniciar o encerramento controlado.
	<-ctx.Done()

	// Concede até dez segundos para que as requisições em andamento sejam concluídas.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
