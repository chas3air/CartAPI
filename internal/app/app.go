package app

import (
	carthandler "cartapi/internal/handlers/cart"
	"cartapi/internal/models"
	cartservice "cartapi/internal/service/cart"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
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

	http.HandleFunc("/carts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			cartItemHandler.CreateCart(w, r)
			return
		}
		http.NotFound(w, r)
	})

	http.HandleFunc("/carts/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.Trim(r.URL.Path, "/")
		parts := strings.Split(path, "/")

		if len(parts) == 2 && r.Method == http.MethodGet {
			// GET /carts/{cartId}
			cartId := parts[1]
			cartItemHandler.ViewCart(w, r, cartId)
			return
		}

		if len(parts) == 3 && parts[2] == "items" && r.Method == http.MethodPost {
			// POST /carts/{cartId}/items
			cartId := parts[1]
			cartItemHandler.AddToCart(w, r, cartId)
			return
		}

		if len(parts) == 4 && parts[2] == "items" && r.Method == http.MethodDelete {
			// DELETE /carts/{cartId}/items/{itemId}
			cartId := parts[1]
			itemId := parts[3]
			cartItemHandler.RemoveFromCart(w, r, cartId, itemId)
			return
		}

		http.NotFound(w, r)
	})

	if err := http.ListenAndServe(
		fmt.Sprintf(":%d", a.port),
		nil,
	); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
