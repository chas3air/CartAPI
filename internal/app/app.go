package app

import (
	carthandler "cartapi/internal/handlers/cart"
	"cartapi/internal/models"
	"cartapi/internal/routes"
	cartservice "cartapi/internal/service/cart"
	"context"
	"fmt"
	"log/slog"
	"net/http"
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
}

func New(log *slog.Logger, port int, storage CartItemStorage) *App {
	return &App{
		log:     log,
		port:    port,
		storage: storage,
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *App) Run() error {
	const op = "app.Run"

	cartItemService := cartservice.New(a.log, a.storage)
	cartItemHandler := carthandler.New(a.log, cartItemService)

	router := routes.New(cartItemHandler)
	router.Register()

	if err := http.ListenAndServe(
		fmt.Sprintf(":%d", a.port),
		nil,
	); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
