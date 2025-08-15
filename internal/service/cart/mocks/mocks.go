package mocks

import (
	"cartapi/internal/models"

	"context"

	"github.com/stretchr/testify/mock"
)

type Service struct {
	mock.Mock
}

func (m *Service) CreateCart(ctx context.Context) (models.Cart, error) {
	args := m.Called(ctx)
	return args.Get(0).(models.Cart), args.Error(1)
}
func (m *Service) AddToCart(ctx context.Context, cartId int, item models.CartItem) (models.CartItem, error) {
	args := m.Called(ctx, cartId, item)
	return args.Get(0).(models.CartItem), args.Error(1)
}
func (m *Service) RemoveFromCart(ctx context.Context, cartId int, itemId int) error {
	args := m.Called(ctx, cartId, itemId)
	return args.Error(0)
}
func (m *Service) ViewCart(ctx context.Context, cartId int) (models.Cart, error) {
	args := m.Called(ctx, cartId)
	return args.Get(0).(models.Cart), args.Error(1)
}
