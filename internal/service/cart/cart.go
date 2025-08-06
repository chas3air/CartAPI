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
		if err := ctx.Err(); err != nil {
			if errors.Is(err, context.Canceled) {
				log.Warn("context canceled", sl.Err(err))
				return models.Cart{}, fmt.Errorf("%s: %w", op, serviceerrors.ErrContextCanceled)
			} else if errors.Is(err, context.DeadlineExceeded) {
				log.Warn("deadline exceeded", sl.Err(err))
				return models.Cart{}, fmt.Errorf("%s: %w", op, serviceerrors.ErrDeadlineExceeded)
			} else {
				log.Error("unexpected error", sl.Err(err))
				return models.Cart{}, fmt.Errorf("%s: %w", op, err)
			}
		}
	default:
	}

	cart, err := c.storage.CreateCart(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Warn("context canceled", sl.Err(serviceerrors.ErrContextCanceled))
			return models.Cart{}, fmt.Errorf("%s: %w", op, serviceerrors.ErrContextCanceled)
		} else if errors.Is(err, context.DeadlineExceeded) {
			log.Warn("deadline exceeded", sl.Err(serviceerrors.ErrDeadlineExceeded))
			return models.Cart{}, fmt.Errorf("%s: %w", op, serviceerrors.ErrDeadlineExceeded)
		} else {
			log.Error("Failed to create a cart", sl.Err(err))
			return models.Cart{}, fmt.Errorf("%s: %w", op, err)
		}
	}

	return cart, nil
}

func (c *CartApiService) AddToCart(ctx context.Context, cartId int, item models.CartItem) (models.CartItem, error) {
	const op = "service.cartapi.AddToCart"
	log := c.log.With("op", op)

	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != nil {
			if errors.Is(err, context.Canceled) {
				log.Warn("context canceled", sl.Err(err))
				return models.CartItem{}, fmt.Errorf("%s: %w", op, serviceerrors.ErrContextCanceled)
			} else if errors.Is(err, context.DeadlineExceeded) {
				log.Warn("deadline exceeded", sl.Err(err))
				return models.CartItem{}, fmt.Errorf("%s: %w", op, serviceerrors.ErrDeadlineExceeded)
			} else {
				log.Error("unexpected error", sl.Err(err))
				return models.CartItem{}, fmt.Errorf("%s: %w", op, err)
			}
		}
	default:
	}

	cartItem, err := c.storage.AddToCart(ctx, cartId, item)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Warn("context canceled", sl.Err(serviceerrors.ErrContextCanceled))
			return models.CartItem{}, fmt.Errorf("%s: %w", op, serviceerrors.ErrContextCanceled)
		} else if errors.Is(err, context.DeadlineExceeded) {
			log.Warn("deadline exceeded", sl.Err(serviceerrors.ErrDeadlineExceeded))
			return models.CartItem{}, fmt.Errorf("%s: %w", op, serviceerrors.ErrDeadlineExceeded)
		} else if errors.Is(err, databaseerrors.ErrNotFound) {
			log.Warn("cart not found", sl.Err(serviceerrors.ErrNotFound))
			return models.CartItem{}, fmt.Errorf("%s: %w", op, serviceerrors.ErrNotFound)
		} else {
			log.Error("Failed to add item to cart", sl.Err(err))
			return models.CartItem{}, fmt.Errorf("%s: %w", op, err)
		}
	}

	return cartItem, nil
}

func (c *CartApiService) RemoveFromCart(ctx context.Context, cartId int, itemId int) error {
	const op = "service.cartapi.RemoveFromCart"
	log := c.log.With("op", op)

	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != nil {
			if errors.Is(err, context.Canceled) {
				log.Warn("context canceled", sl.Err(err))
				return fmt.Errorf("%s: %w", op, serviceerrors.ErrContextCanceled)
			} else if errors.Is(err, context.DeadlineExceeded) {
				log.Warn("deadline exceeded", sl.Err(err))
				return fmt.Errorf("%s: %w", op, serviceerrors.ErrDeadlineExceeded)
			} else {
				log.Error("unexpected error", sl.Err(err))
				return fmt.Errorf("%s: %w", op, err)
			}
		}
	default:
	}

	err := c.storage.RemoveFromCart(ctx, cartId, itemId)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Warn("context canceled", sl.Err(serviceerrors.ErrContextCanceled))
			return fmt.Errorf("%s: %w", op, serviceerrors.ErrContextCanceled)
		} else if errors.Is(err, context.DeadlineExceeded) {
			log.Warn("deadline exceeded", sl.Err(serviceerrors.ErrDeadlineExceeded))
			return fmt.Errorf("%s: %w", op, serviceerrors.ErrDeadlineExceeded)
		} else if errors.Is(err, databaseerrors.ErrNotFound) {
			log.Warn("cart or item doesn't exist", sl.Err(serviceerrors.ErrNotFound))
			return fmt.Errorf("%s: %w", op, serviceerrors.ErrNotFound)
		} else {
			log.Error("Failed to remove item from cart", sl.Err(err))
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	return nil
}

func (c *CartApiService) ViewCart(ctx context.Context, cartId int) (models.Cart, error) {
	const op = "service.cartapi.ViewCart"
	log := c.log.With("op", op)

	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != nil {
			if errors.Is(err, context.Canceled) {
				log.Warn("context canceled", sl.Err(err))
				return models.Cart{}, fmt.Errorf("%s: %w", op, serviceerrors.ErrContextCanceled)
			} else if errors.Is(err, context.DeadlineExceeded) {
				log.Warn("deadline exceeded", sl.Err(err))
				return models.Cart{}, fmt.Errorf("%s: %w", op, serviceerrors.ErrDeadlineExceeded)
			} else {
				log.Error("unexpected error", sl.Err(err))
				return models.Cart{}, fmt.Errorf("%s: %w", op, err)
			}
		}
	default:
	}

	cart, err := c.storage.ViewCart(ctx, cartId)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Warn("context canceled", sl.Err(serviceerrors.ErrContextCanceled))
			return models.Cart{}, fmt.Errorf("%s: %w", op, serviceerrors.ErrContextCanceled)
		} else if errors.Is(err, context.DeadlineExceeded) {
			log.Warn("deadline exceeded", sl.Err(serviceerrors.ErrDeadlineExceeded))
			return models.Cart{}, fmt.Errorf("%s: %w", op, serviceerrors.ErrDeadlineExceeded)
		} else {
			log.Error("Failed to get items from cart", sl.Err(err))
			return models.Cart{}, fmt.Errorf("%s: %w", op, err)
		}
	}

	return cart, nil
}
