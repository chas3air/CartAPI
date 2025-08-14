package cartservice_test

import (
	databaseerrors "cartapi/internal/database"
	"cartapi/internal/handlers/cart/mocks"
	"cartapi/internal/models"
	serviceerrors "cartapi/internal/service"
	cartservice "cartapi/internal/service/cart"
	"cartapi/pkg/lib/logger/slogdiscard"

	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestService(storage *mocks.Service) *cartservice.CartApiService {
	logger := slogdiscard.NewDiscardLogger()
	return cartservice.New(logger, storage)
}

func TestContextCanceled(t *testing.T) {
	t.Run("CreateCart context canceled before call", func(t *testing.T) {
		mockStorage := new(mocks.Service)
		svc := newTestService(mockStorage)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := svc.CreateCart(ctx)
		assert.ErrorIs(t, err, serviceerrors.ErrContextCanceled)

		mockStorage.AssertExpectations(t)
	})

	t.Run("AddToCart context canceled before call", func(t *testing.T) {
		mockStorage := new(mocks.Service)
		svc := newTestService(mockStorage)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		cartId := 1
		item := models.CartItem{}

		_, err := svc.AddToCart(ctx, cartId, item)
		assert.ErrorIs(t, err, serviceerrors.ErrContextCanceled)

		mockStorage.AssertExpectations(t)
	})

	t.Run("RemoveFromCart context canceled before call", func(t *testing.T) {
		mockStorage := new(mocks.Service)
		svc := newTestService(mockStorage)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := svc.RemoveFromCart(ctx, 1, 1)
		assert.ErrorIs(t, err, serviceerrors.ErrContextCanceled)

		mockStorage.AssertExpectations(t)
	})

	t.Run("ViewCart context canceled before call", func(t *testing.T) {
		mockStorage := new(mocks.Service)
		svc := newTestService(mockStorage)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := svc.ViewCart(ctx, 1)
		assert.ErrorIs(t, err, serviceerrors.ErrContextCanceled)

		mockStorage.AssertExpectations(t)
	})

}

func TestDeadlineExceeded(t *testing.T) {
	t.Run("CreateCart context deadline exceeded", func(t *testing.T) {
		mockStorage := new(mocks.Service)
		svc := newTestService(mockStorage)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		time.Sleep(time.Millisecond * 15)

		_, err := svc.CreateCart(ctx)
		assert.ErrorIs(t, err, serviceerrors.ErrDeadlineExceeded)

		mockStorage.AssertExpectations(t)
	})

	t.Run("AddToCart context deadline exceeded", func(t *testing.T) {
		mockStorage := new(mocks.Service)
		svc := newTestService(mockStorage)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		time.Sleep(time.Millisecond * 15)

		cartId := 1
		item := models.CartItem{}

		_, err := svc.AddToCart(ctx, cartId, item)
		assert.ErrorIs(t, err, serviceerrors.ErrDeadlineExceeded)

		mockStorage.AssertExpectations(t)
	})

	t.Run("RemoveFromCart context deadline exceeded", func(t *testing.T) {
		mockStorage := new(mocks.Service)
		svc := newTestService(mockStorage)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		time.Sleep(time.Millisecond * 15)

		err := svc.RemoveFromCart(ctx, 1, 1)
		assert.ErrorIs(t, err, serviceerrors.ErrDeadlineExceeded)

		mockStorage.AssertExpectations(t)
	})

	t.Run("ViewCart context deadline exceeded", func(t *testing.T) {
		mockStorage := new(mocks.Service)
		svc := newTestService(mockStorage)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()
		time.Sleep(time.Millisecond * 15)

		_, err := svc.ViewCart(ctx, 1)
		assert.ErrorIs(t, err, serviceerrors.ErrDeadlineExceeded)

		mockStorage.AssertExpectations(t)
	})
}

func TestCreateCart(t *testing.T) {
	tests := []struct {
		name       string
		mockReturn func(*mocks.Service)
		wantCart   models.Cart
		wantErr    bool
	}{
		{
			name: "Success",
			mockReturn: func(s *mocks.Service) {
				s.On("CreateCart", mock.Anything).Return(models.Cart{}, nil)
			},
			wantCart: models.Cart{},
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(mocks.Service)
			tt.mockReturn(mockStorage)
			svc := newTestService(mockStorage)

			got, err := svc.CreateCart(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCart, got)
			}
			mockStorage.AssertExpectations(t)
		})
	}
}

func TestAddToCart(t *testing.T) {
	tests := []struct {
		name       string
		cartId     int
		item       models.CartItem
		mockReturn func(*mocks.Service)
		wantItem   models.CartItem
		wantErr    bool
		errType    error
	}{
		{
			name:   "Success",
			cartId: 1,
			item: models.CartItem{
				Id:       1,
				CartId:   1,
				Product:  "item",
				Quantity: 10,
			},
			mockReturn: func(s *mocks.Service) {
				s.On("AddToCart", mock.Anything, 1, mock.Anything).Return(models.CartItem{
					Id:       1,
					CartId:   1,
					Product:  "item",
					Quantity: 10,
				}, nil)
			},
			wantItem: models.CartItem{
				Id:       1,
				CartId:   1,
				Product:  "item",
				Quantity: 10,
			},
			wantErr: false,
		},
		{
			name:   "NotFound error",
			cartId: 1,
			item: models.CartItem{
				Id:       1,
				CartId:   1,
				Product:  "item",
				Quantity: 10,
			},
			mockReturn: func(s *mocks.Service) {
				s.On("AddToCart", mock.Anything, 1, mock.Anything).Return(models.CartItem{}, databaseerrors.ErrNotFound)
			},
			wantErr: true,
			errType: serviceerrors.ErrNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(mocks.Service)
			tt.mockReturn(mockStorage)
			svc := newTestService(mockStorage)

			got, err := svc.AddToCart(context.Background(), tt.cartId, tt.item)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantItem, got)
			}
			mockStorage.AssertExpectations(t)
		})
	}
}

func TestRemoveFromCart(t *testing.T) {
	tests := []struct {
		name       string
		cartId     int
		itemId     int
		mockReturn func(*mocks.Service)
		wantErr    bool
		errType    error
	}{
		{
			name:   "Success",
			cartId: 1,
			itemId: 1,
			mockReturn: func(s *mocks.Service) {
				s.On("RemoveFromCart", mock.Anything, 1, 1).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "NotFound error",
			cartId: 1,
			itemId: 1,
			mockReturn: func(s *mocks.Service) {
				s.On("RemoveFromCart", mock.Anything, 1, 1).Return(databaseerrors.ErrNotFound)
			},
			wantErr: true,
			errType: serviceerrors.ErrNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(mocks.Service)
			tt.mockReturn(mockStorage)
			svc := newTestService(mockStorage)

			err := svc.RemoveFromCart(context.Background(), tt.cartId, tt.itemId)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
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
		name       string
		cartId     int
		mockReturn func(*mocks.Service)
		wantCart   models.Cart
		wantErr    bool
		errType    error
	}{
		{
			name:   "Success",
			cartId: 1,
			mockReturn: func(s *mocks.Service) {
				items := []models.CartItem{
					{Id: 2, CartId: 1, Product: "item", Quantity: 3},
				}
				s.On("ViewCart", mock.Anything, 1).Return(models.Cart{Id: 1, Items: items}, nil)
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
			name:   "NotFound error",
			cartId: 1,
			mockReturn: func(s *mocks.Service) {
				s.On("ViewCart", mock.Anything, 1).Return(models.Cart{}, databaseerrors.ErrNotFound)
			},
			wantErr: true,
			errType: serviceerrors.ErrNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(mocks.Service)
			tt.mockReturn(mockStorage)
			svc := newTestService(mockStorage)

			got, err := svc.ViewCart(context.Background(), tt.cartId)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCart, got)
			}
			mockStorage.AssertExpectations(t)
		})
	}
}
