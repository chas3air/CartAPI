package app

import (
	"cartapi/internal/database/psql"
	carthandler "cartapi/internal/handlers/cart"
	"cartapi/internal/routes"
	cartservice "cartapi/internal/service/cart"
	"cartapi/pkg/config"
	"cartapi/pkg/lib/logger"
	"cartapi/pkg/lib/logger/sl"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Run() error {
	const op = "app.Run"

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log, err := logger.SetupLogger(cfg.HTTP.Env)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	storage, err := psql.New(log, cfg.ConnectionString())
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	cartItemService := cartservice.New(log, storage)
	cartItemHandler := carthandler.New(log, cartItemService)

	router := routes.New(cartItemHandler)
	router.Register()

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler: nil,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Server failed to start", sl.Err(err))
		}
	}()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGTERM, syscall.SIGINT)
	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("Failed to shutdown server", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	} else {
		log.Info("Server shutdown gracefully")
	}

	if err := storage.Close(); err != nil {
		log.Error("Failed to close database connection", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	} else {
		log.Info("Database connection closed gracefully")
	}

	return nil
}
