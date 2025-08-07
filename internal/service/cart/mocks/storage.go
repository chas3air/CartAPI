package mocks

import (
	"cartapi/internal/models"
	"context"

	"github.com/stretchr/testify/mock"
)

type Storage struct {
	mock.Mock
}

func (m *Storage) CreateCart(ctx context.Context) (models.Cart, error) {
	args := m.Called(ctx)
	return args.Get(0).(models.Cart), args.Error(1)
}
func (m *Storage) AddToCart(ctx context.Context, cartId int, item models.CartItem) (models.CartItem, error) {
	args := m.Called(ctx, cartId, item)
	return args.Get(0).(models.CartItem), args.Error(1)
}
func (m *Storage) RemoveFromCart(ctx context.Context, cartId int, itemId int) error {
	args := m.Called(ctx, cartId, itemId)
	return args.Error(0)
}
func (m *Storage) ViewCart(ctx context.Context, cartId int) (models.Cart, error) {
	args := m.Called(ctx, cartId)
	return args.Get(0).(models.Cart), args.Error(1)
}
