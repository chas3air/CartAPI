package main

import (
	"cartapi/internal/app"
	"cartapi/internal/database/psql"
	"cartapi/pkg/config"
	"cartapi/pkg/lib/logger"
	"os"
	"os/signal"
	"sync"
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

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		application.MustRun()
	}()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGTERM, syscall.SIGINT)
	<-done

	log.Info("Closing database")
	storage.Close()

	wg.Wait()
}
