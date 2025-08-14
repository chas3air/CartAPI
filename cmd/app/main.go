package main

import (
	"cartapi/internal/app"
	"cartapi/internal/database/psql"
	"cartapi/pkg/config"
	"cartapi/pkg/lib/logger"
	"cartapi/pkg/lib/logger/sl"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log, err := logger.SetupLogger(cfg.HTTP.Env)
	if err != nil {
		panic(err)
	}

	storage, err := psql.New(log, cfg.ConnectionString())
	if err != nil {
		panic(err)
	}

	application := app.New(
		log,
		cfg.HTTP.Port,
		storage,
	)

	go func() {
		if err := application.Run(); err != nil {
			log.Error("Application failed to start", sl.Err(err))
			panic(err)
		}
	}()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGTERM, syscall.SIGINT)
	<-done

	log.Info("Closing database")
	storage.Close()
}
