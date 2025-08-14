package app

import (
	carthandler "cartapi/internal/handlers/cart"
	"cartapi/internal/models"
	"cartapi/internal/routes"
	cartservice "cartapi/internal/service/cart"
	"cartapi/pkg/lib/logger/sl"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type CartItemStorage interface {
	CreateCart(ctx context.Context) (models.Cart, error)
	AddToCart(ctx context.Context, cartId int, item models.CartItem) (models.CartItem, error)
	RemoveFromCart(ctx context.Context, cartId int, itemId int) error
	ViewCart(ctx context.Context, cartId int) (models.Cart, error)
}

type App struct {
	log     *slog.Logger
	port    int
	storage CartItemStorage
	server  *http.Server
}

func New(log *slog.Logger, port int, storage CartItemStorage) *App {
	return &App{
		log:     log,
		port:    port,
		storage: storage,
	}
}

func (a *App) Run() error {
	const op = "app.Run"

	cartItemService := cartservice.New(a.log, a.storage)
	cartItemHandler := carthandler.New(a.log, cartItemService)

	router := routes.New(cartItemHandler)
	router.Register()

	a.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", a.port),
		Handler: nil,
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.log.Error("Server failed to start", sl.Err(err))
		}
	}()

	<-signalChan
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		a.log.Error("failed to shutdown server", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
