package cartservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	databaseerrors "cartapi/internal/database"
	"cartapi/internal/models"
	serviceerrors "cartapi/internal/service"
	"cartapi/pkg/lib/logger/sl"
)

type CartItemStorage interface {
	CreateCart(ctx context.Context) (models.Cart, error)
	AddToCart(ctx context.Context, cartId int, item models.CartItem) (models.CartItem, error)
	RemoveFromCart(ctx context.Context, cartId int, itemId int) error
	ViewCart(ctx context.Context, cartId int) (models.Cart, error)
}

type CartApiService struct {
	log     *slog.Logger
	storage CartItemStorage
}

func New(log *slog.Logger, storage CartItemStorage) *CartApiService {
	return &CartApiService{
		log:     log,
		storage: storage,
	}
}

func (c *CartApiService) CreateCart(ctx context.Context) (models.Cart, error) {
	const op = "service.cartapi.CreateCart"
	log := c.log.With("op", op)

	select {
	case <-ctx.Done():
		return models.Cart{}, handleContextError(log, ctx, op)
	default:
	}

	cart, err := c.storage.CreateCart(ctx)
	if err != nil {
		return models.Cart{}, handleDatabaseError(log, err, op, "Failed to create a cart")
	}

	return cart, nil
}

func (c *CartApiService) AddToCart(ctx context.Context, cartId int, item models.CartItem) (models.CartItem, error) {
	const op = "service.cartapi.AddToCart"
	log := c.log.With("op", op)

	select {
	case <-ctx.Done():
		return models.CartItem{}, handleContextError(log, ctx, op)
	default:
	}

	cartItem, err := c.storage.AddToCart(ctx, cartId, item)
	if err != nil {
		return models.CartItem{}, handleDatabaseError(log, err, op, "Failed to add item to cart")
	}

	return cartItem, nil
}

func (c *CartApiService) RemoveFromCart(ctx context.Context, cartId int, itemId int) error {
	const op = "service.cartapi.RemoveFromCart"
	log := c.log.With("op", op)

	select {
	case <-ctx.Done():
		return handleContextError(log, ctx, op)
	default:
	}

	err := c.storage.RemoveFromCart(ctx, cartId, itemId)
	if err != nil {
		return handleDatabaseError(log, err, op, "Failed to remove item from cart")
	}

	return nil
}

func (c *CartApiService) ViewCart(ctx context.Context, cartId int) (models.Cart, error) {
	const op = "service.cartapi.ViewCart"
	log := c.log.With("op", op)

	select {
	case <-ctx.Done():
		return models.Cart{}, handleContextError(log, ctx, op)
	default:
	}

	cart, err := c.storage.ViewCart(ctx, cartId)
	if err != nil {
		return models.Cart{}, handleDatabaseError(log, err, op, "Failed to get items from cart")
	}

	return cart, nil
}

func handleContextError(log *slog.Logger, ctx context.Context, op string) error {
	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			log.Warn("context canceled", sl.Err(err))
			return fmt.Errorf("%s: %w", op, serviceerrors.ErrContextCanceled)
		} else if errors.Is(err, context.DeadlineExceeded) {
			log.Warn("deadline exceeded", sl.Err(err))
			return fmt.Errorf("%s: %w", op, serviceerrors.ErrDeadlineExceeded)
		}
	}

	return nil
}

func handleDatabaseError(log *slog.Logger, err error, op string, msg string) error {
	if errors.Is(err, context.Canceled) {
		log.Warn("context canceled", sl.Err(serviceerrors.ErrContextCanceled))
		return fmt.Errorf("%s: %w", op, serviceerrors.ErrContextCanceled)
	} else if errors.Is(err, context.DeadlineExceeded) {
		log.Warn("deadline exceeded", sl.Err(serviceerrors.ErrDeadlineExceeded))
		return fmt.Errorf("%s: %w", op, serviceerrors.ErrDeadlineExceeded)
	} else if errors.Is(err, databaseerrors.ErrNotFound) {
		log.Warn("cart not found", sl.Err(serviceerrors.ErrNotFound))
		return fmt.Errorf("%s: %w", op, serviceerrors.ErrNotFound)
	} else {
		log.Error(msg, sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}
}
