package main

import (
	"cartapi/internal/app"
	"cartapi/internal/database/psql"
	"cartapi/pkg/config"
	"cartapi/pkg/lib/logger"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.MustLoad()

	log := logger.SetupLogger(cfg.HTTP.Env)

	storage := psql.New(log, cfg.ConnectionString())

	application := app.New(
		log,
		cfg.HTTP.Port,
		storage,
	)

	go func() {
		application.MustRun()
	}()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGTERM, syscall.SIGINT)
	<-done

	log.Info("Closing database")
	storage.Close()
}
