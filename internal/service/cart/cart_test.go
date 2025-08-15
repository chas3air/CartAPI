package cartservice_test

import (
	"context"
	"errors"
	"testing"

	databaseerrors "cartapi/internal/database"
	"cartapi/internal/handlers/cart/mocks"
	"cartapi/internal/models"
	serviceerrors "cartapi/internal/service"
	cartservice "cartapi/internal/service/cart"
	"cartapi/pkg/lib/logger/slogdiscard"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestService(storage *mocks.Service) *cartservice.CartApiService {
	logger := slogdiscard.NewDiscardLogger()
	return cartservice.New(logger, storage)
}

func TestCreateCart(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(s *mocks.Service)
		wantCart  models.Cart
		wantErr   bool
		errType   error
	}{
		{
			name: "Success",
			mockSetup: func(s *mocks.Service) {
				s.On("CreateCart", mock.Anything).Return(models.Cart{}, nil)
			},
			wantCart: models.Cart{},
			wantErr:  false,
		},
		{
			name: "Context canceled error",
			mockSetup: func(s *mocks.Service) {
				s.On("CreateCart", mock.Anything).Return(models.Cart{}, serviceerrors.ErrContextCanceled)
			},
			wantErr: true,
			errType: serviceerrors.ErrContextCanceled,
		},
		{
			name: "Deadline exceeded error",
			mockSetup: func(s *mocks.Service) {
				s.On("CreateCart", mock.Anything).Return(models.Cart{}, serviceerrors.ErrDeadlineExceeded)
			},
			wantErr: true,
			errType: serviceerrors.ErrDeadlineExceeded,
		},
		{
			name: "Generic error",
			mockSetup: func(s *mocks.Service) {
				s.On("CreateCart", mock.Anything).Return(models.Cart{}, errors.New("error"))
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockStorage := new(mocks.Service)
			tc.mockSetup(mockStorage)
			svc := newTestService(mockStorage)

			got, err := svc.CreateCart(context.Background())
			if tc.wantErr {
				assert.Error(t, err)
				if tc.errType != nil {
					assert.ErrorIs(t, err, tc.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantCart, got)
			}
			mockStorage.AssertExpectations(t)
		})
	}
}

func TestAddToCart(t *testing.T) {
	tests := []struct {
		name      string
		cartId    int
		item      models.CartItem
		mockSetup func(s *mocks.Service)
		wantItem  models.CartItem
		wantErr   bool
		errType   error
	}{
		{
			name:   "Success",
			cartId: 1,
			item:   models.CartItem{Id: 1, CartId: 1, Product: "item", Quantity: 10},
			mockSetup: func(s *mocks.Service) {
				s.On("AddToCart", mock.Anything, 1, mock.Anything).Return(models.CartItem{
					Id:       1,
					CartId:   1,
					Product:  "item",
					Quantity: 10,
				}, nil)
			},
			wantItem: models.CartItem{Id: 1, CartId: 1, Product: "item", Quantity: 10},
			wantErr:  false,
		},
		{
			name:   "Context canceled error",
			cartId: 1,
			item:   models.CartItem{},
			mockSetup: func(s *mocks.Service) {
				s.On("AddToCart", mock.Anything, 1, mock.Anything).Return(models.CartItem{}, serviceerrors.ErrContextCanceled)
			},
			wantErr: true,
			errType: serviceerrors.ErrContextCanceled,
		},
		{
			name:   "Deadline exceeded error",
			cartId: 1,
			item:   models.CartItem{},
			mockSetup: func(s *mocks.Service) {
				s.On("AddToCart", mock.Anything, 1, mock.Anything).Return(models.CartItem{}, serviceerrors.ErrDeadlineExceeded)
			},
			wantErr: true,
			errType: serviceerrors.ErrDeadlineExceeded,
		},
		{
			name:   "NotFound error",
			cartId: 1,
			item:   models.CartItem{Id: 1, CartId: 1, Product: "item", Quantity: 10},
			mockSetup: func(s *mocks.Service) {
				s.On("AddToCart", mock.Anything, 1, mock.Anything).Return(models.CartItem{}, databaseerrors.ErrNotFound)
			},
			wantErr: true,
			errType: serviceerrors.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockStorage := new(mocks.Service)
			tc.mockSetup(mockStorage)
			svc := newTestService(mockStorage)

			got, err := svc.AddToCart(context.Background(), tc.cartId, tc.item)
			if tc.wantErr {
				assert.Error(t, err)
				if tc.errType != nil {
					assert.ErrorIs(t, err, tc.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantItem, got)
			}
			mockStorage.AssertExpectations(t)
		})
	}
}

func TestRemoveFromCart(t *testing.T) {
	tests := []struct {
		name      string
		cartId    int
		itemId    int
		mockSetup func(s *mocks.Service)
		wantErr   bool
		errType   error
	}{
		{
			name:   "Success",
			cartId: 1,
			itemId: 1,
			mockSetup: func(s *mocks.Service) {
				s.On("RemoveFromCart", mock.Anything, 1, 1).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "Context canceled error",
			cartId: 1,
			itemId: 1,
			mockSetup: func(s *mocks.Service) {
				s.On("RemoveFromCart", mock.Anything, 1, 1).Return(serviceerrors.ErrContextCanceled)
			},
			wantErr: true,
			errType: serviceerrors.ErrContextCanceled,
		},
		{
			name:   "Deadline exceeded error",
			cartId: 1,
			itemId: 1,
			mockSetup: func(s *mocks.Service) {
				s.On("RemoveFromCart", mock.Anything, 1, 1).Return(serviceerrors.ErrDeadlineExceeded)
			},
			wantErr: true,
			errType: serviceerrors.ErrDeadlineExceeded,
		},
		{
			name:   "NotFound error",
			cartId: 1,
			itemId: 1,
			mockSetup: func(s *mocks.Service) {
				s.On("RemoveFromCart", mock.Anything, 1, 1).Return(databaseerrors.ErrNotFound)
			},
			wantErr: true,
			errType: serviceerrors.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockStorage := new(mocks.Service)
			tc.mockSetup(mockStorage)
			svc := newTestService(mockStorage)

			err := svc.RemoveFromCart(context.Background(), tc.cartId, tc.itemId)
			if tc.wantErr {
				assert.Error(t, err)
				if tc.errType != nil {
					assert.ErrorIs(t, err, tc.errType)
				}
			} else {
				assert.NoError(t, err)
			}
			mockStorage.AssertExpectations(t)
		})
	}
}

func TestViewCart(t *testing.T) {
	tests := []struct {
		name      string
		cartId    int
		mockSetup func(s *mocks.Service)
		wantCart  models.Cart
		wantErr   bool
		errType   error
	}{
		{
			name:   "Success",
			cartId: 1,
			mockSetup: func(s *mocks.Service) {
				s.On("ViewCart", mock.Anything, 1).Return(models.Cart{
					Id: 1,
					Items: []models.CartItem{
						{Id: 2, CartId: 1, Product: "item", Quantity: 3},
					},
				}, nil)
			},
			wantCart: models.Cart{
				Id: 1,
				Items: []models.CartItem{
					{Id: 2, CartId: 1, Product: "item", Quantity: 3},
				},
			},
			wantErr: false,
		},
		{
			name:   "Context canceled error",
			cartId: 1,
			mockSetup: func(s *mocks.Service) {
				s.On("ViewCart", mock.Anything, 1).Return(models.Cart{}, serviceerrors.ErrContextCanceled)
			},
			wantErr: true,
			errType: serviceerrors.ErrContextCanceled,
		},
		{
			name:   "Deadline exceeded error",
			cartId: 1,
			mockSetup: func(s *mocks.Service) {
				s.On("ViewCart", mock.Anything, 1).Return(models.Cart{}, serviceerrors.ErrDeadlineExceeded)
			},
			wantErr: true,
			errType: serviceerrors.ErrDeadlineExceeded,
		},
		{
			name:   "NotFound error",
			cartId: 1,
			mockSetup: func(s *mocks.Service) {
				s.On("ViewCart", mock.Anything, 1).Return(models.Cart{}, databaseerrors.ErrNotFound)
			},
			wantErr: true,
			errType: serviceerrors.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockStorage := new(mocks.Service)
			tc.mockSetup(mockStorage)
			svc := newTestService(mockStorage)

			got, err := svc.ViewCart(context.Background(), tc.cartId)
			if tc.wantErr {
				assert.Error(t, err)
				if tc.errType != nil {
					assert.ErrorIs(t, err, tc.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantCart, got)
			}
			mockStorage.AssertExpectations(t)
		})
	}
}
