package cartservice_test

import (
	databaseerrors "cartapi/internal/database"
	"cartapi/internal/models"
	serviceerrors "cartapi/internal/service"
	cartservice "cartapi/internal/service/cart"
	"cartapi/internal/service/cart/mocks"
	"cartapi/pkg/lib/logger/handler/slogdiscard"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestService(storage *mocks.MockStorage) *cartservice.CartApiService {
	logger := slogdiscard.NewDiscardLogger()
	return cartservice.New(logger, storage)
}
func TestCreateCart(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)
		cart := models.Cart{}
		mockStorage.On("CreateCart", mock.Anything).Return(models.Cart{}, nil)

		svc := newTestService(mockStorage)
		got, err := svc.CreateCart(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, cart, got)

		mockStorage.AssertExpectations(t)
	})

	t.Run("ContextCanceled", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)
		svc := newTestService(mockStorage)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := svc.CreateCart(ctx)
		assert.Error(t, err, serviceerrors.ErrContextCanceled)
		assert.Contains(t, err.Error(), "context canceled")

		mockStorage.AssertExpectations(t)
	})

	t.Run("DeadlineExceeded", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)
		svc := newTestService(mockStorage)

		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*10)
		time.Sleep(time.Millisecond * 15)

		_, err := svc.CreateCart(ctx)
		assert.Error(t, err, serviceerrors.ErrDeadlineExceeded)
		assert.Contains(t, err.Error(), "deadline exceeded")

		mockStorage.AssertExpectations(t)
	})

	t.Run("Failed to create a cart", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)
		mockStorage.On("CreateCart", mock.Anything).Return(models.Cart{}, errors.New("error"))

		svc := newTestService(mockStorage)
		_, err := svc.CreateCart(context.Background())
		assert.Error(t, err)

		mockStorage.AssertExpectations(t)
	})

	t.Run("DB context canceled", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)

		mockStorage.On("CreateCart", mock.Anything).Return(models.Cart{}, context.Canceled)

		svc := newTestService(mockStorage)

		_, err := svc.CreateCart(context.TODO())
		assert.ErrorIs(t, err, serviceerrors.ErrContextCanceled)

		mockStorage.AssertExpectations(t)
	})

	t.Run("DB deadline exceeded", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)

		mockStorage.On("CreateCart", mock.Anything).Return(models.Cart{}, context.DeadlineExceeded)

		svc := newTestService(mockStorage)

		_, err := svc.CreateCart(context.TODO())
		assert.ErrorIs(t, err, serviceerrors.ErrDeadlineExceeded)

		mockStorage.AssertExpectations(t)
	})
}

func TestAddToCart(t *testing.T) {
	t.Run("ContextCanceled", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)
		svc := newTestService(mockStorage)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		cartId := 0
		item := models.CartItem{}
		_, err := svc.AddToCart(ctx, cartId, item)
		assert.Error(t, err, serviceerrors.ErrContextCanceled)
		assert.Contains(t, err.Error(), "context canceled")

		mockStorage.AssertExpectations(t)
	})

	t.Run("DeadlineExceeded", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)
		svc := newTestService(mockStorage)

		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*10)
		time.Sleep(time.Millisecond * 15)

		cartId := 0
		item := models.CartItem{}
		_, err := svc.AddToCart(ctx, cartId, item)
		assert.Error(t, err, serviceerrors.ErrDeadlineExceeded)
		assert.Contains(t, err.Error(), "deadline exceeded")

		mockStorage.AssertExpectations(t)
	})

	t.Run("Success", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)

		cartId := 0
		item := models.CartItem{
			Id:       1,
			CartId:   cartId,
			Product:  "item",
			Quantity: 10,
		}
		mockStorage.On("AddToCart", mock.Anything, cartId, item).Return(item, nil)

		svc := newTestService(mockStorage)
		got, err := svc.AddToCart(context.Background(), cartId, item)
		assert.Nil(t, err)
		assert.Equal(t, got, item)

		mockStorage.AssertExpectations(t)
	})

	t.Run("DB context canceled", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)

		cartId := 0
		item := models.CartItem{
			Id:       1,
			CartId:   cartId,
			Product:  "item",
			Quantity: 10,
		}

		mockStorage.On("AddToCart", mock.Anything, cartId, item).Return(models.CartItem{}, context.Canceled)

		svc := newTestService(mockStorage)

		_, err := svc.AddToCart(context.TODO(), cartId, item)
		assert.ErrorIs(t, err, serviceerrors.ErrContextCanceled)

		mockStorage.AssertExpectations(t)
	})

	t.Run("DB deadline exceeded", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)

		cartId := 0
		item := models.CartItem{
			Id:       1,
			CartId:   cartId,
			Product:  "item",
			Quantity: 10,
		}

		mockStorage.On("AddToCart", mock.Anything, cartId, item).Return(models.CartItem{}, context.DeadlineExceeded)

		svc := newTestService(mockStorage)

		_, err := svc.AddToCart(context.TODO(), cartId, item)
		assert.ErrorIs(t, err, serviceerrors.ErrDeadlineExceeded)

		mockStorage.AssertExpectations(t)
	})

	t.Run("DB not found", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)

		cartId := 0
		item := models.CartItem{
			Id:       1,
			CartId:   cartId,
			Product:  "item",
			Quantity: 10,
		}

		mockStorage.On("AddToCart", mock.Anything, cartId, item).Return(models.CartItem{}, databaseerrors.ErrNotFound)

		svc := newTestService(mockStorage)

		_, err := svc.AddToCart(context.TODO(), cartId, item)
		assert.ErrorIs(t, err, serviceerrors.ErrNotFound)

		mockStorage.AssertExpectations(t)
	})
	t.Run("DB unexpected error", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)

		cartId := 0
		item := models.CartItem{
			Id:       1,
			CartId:   cartId,
			Product:  "item",
			Quantity: 10,
		}

		mockStorage.On("AddToCart", mock.Anything, cartId, item).Return(models.CartItem{}, errors.New("error"))

		svc := newTestService(mockStorage)

		_, err := svc.AddToCart(context.TODO(), cartId, item)
		assert.NotNil(t, err)

		mockStorage.AssertExpectations(t)
	})
}

func TestRemoveFromCart(t *testing.T) {
	t.Run("Context canceled", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		svc := newTestService(mockStorage)

		err := svc.RemoveFromCart(ctx, 0, 1)
		assert.ErrorIs(t, err, serviceerrors.ErrContextCanceled)

		mockStorage.AssertExpectations(t)
	})

	t.Run("Deadline exceeded", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*10)

		time.Sleep(time.Millisecond * 11)

		svc := newTestService(mockStorage)

		err := svc.RemoveFromCart(ctx, 0, 1)
		assert.ErrorIs(t, err, serviceerrors.ErrDeadlineExceeded)

		mockStorage.AssertExpectations(t)
	})

	t.Run("DB context canceled", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)

		cartId := 0
		itemId := 1
		mockStorage.On("RemoveFromCart", mock.Anything, cartId, itemId).Return(context.Canceled)

		svc := newTestService(mockStorage)

		err := svc.RemoveFromCart(context.TODO(), cartId, itemId)
		assert.ErrorIs(t, err, serviceerrors.ErrContextCanceled)

		mockStorage.AssertExpectations(t)
	})

	t.Run("DB deadline exceeded", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)

		cartId := 0
		itemId := 0

		mockStorage.On("RemoveFromCart", mock.Anything, cartId, itemId).Return(context.DeadlineExceeded)

		svc := newTestService(mockStorage)

		err := svc.RemoveFromCart(context.TODO(), cartId, itemId)
		assert.ErrorIs(t, err, serviceerrors.ErrDeadlineExceeded)

		mockStorage.AssertExpectations(t)
	})

	t.Run("DB not found", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)

		cartId := 0
		itemId := 0

		mockStorage.On("RemoveFromCart", mock.Anything, cartId, itemId).Return(databaseerrors.ErrNotFound)

		svc := newTestService(mockStorage)

		err := svc.RemoveFromCart(context.TODO(), cartId, itemId)
		assert.ErrorIs(t, err, serviceerrors.ErrNotFound)

		mockStorage.AssertExpectations(t)
	})

	t.Run("DB unexpected error", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)

		cartId := 0
		itemId := 0

		mockStorage.On("RemoveFromCart", mock.Anything, cartId, itemId).Return(errors.New("error"))

		svc := newTestService(mockStorage)

		err := svc.RemoveFromCart(context.TODO(), cartId, itemId)
		assert.NotNil(t, err)

		mockStorage.AssertExpectations(t)
	})

	t.Run("Success", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)
		mockStorage.On("RemoveFromCart", mock.Anything, 0, 1).Return(nil)

		svc := newTestService(mockStorage)

		err := svc.RemoveFromCart(context.TODO(), 0, 1)
		assert.Nil(t, err)

		mockStorage.AssertExpectations(t)
	})
}

func TestViewCart(t *testing.T) {
	t.Run("Context canceled", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		svc := newTestService(mockStorage)

		_, err := svc.ViewCart(ctx, 0)
		assert.ErrorIs(t, err, serviceerrors.ErrContextCanceled)

		mockStorage.AssertExpectations(t)
	})

	t.Run("Deadline exceeded", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*10)

		time.Sleep(time.Millisecond * 11)

		svc := newTestService(mockStorage)

		_, err := svc.ViewCart(ctx, 0)
		assert.ErrorIs(t, err, serviceerrors.ErrDeadlineExceeded)

		mockStorage.AssertExpectations(t)
	})

	t.Run("DB context canceled", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)

		cartId := 0
		mockStorage.On("ViewCart", mock.Anything, cartId).Return(models.Cart{}, context.Canceled)

		svc := newTestService(mockStorage)

		_, err := svc.ViewCart(context.TODO(), cartId)
		assert.ErrorIs(t, err, serviceerrors.ErrContextCanceled)

		mockStorage.AssertExpectations(t)
	})

	t.Run("DB deadline exceeded", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)

		cartId := 0

		mockStorage.On("ViewCart", mock.Anything, cartId).Return(models.Cart{}, context.DeadlineExceeded)

		svc := newTestService(mockStorage)

		_, err := svc.ViewCart(context.TODO(), cartId)
		assert.ErrorIs(t, err, serviceerrors.ErrDeadlineExceeded)

		mockStorage.AssertExpectations(t)
	})

	t.Run("DB unexpected error", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)

		cartId := 0

		mockStorage.On("ViewCart", mock.Anything, cartId).Return(models.Cart{}, errors.New("error"))

		svc := newTestService(mockStorage)

		_, err := svc.ViewCart(context.TODO(), cartId)
		assert.NotNil(t, err)

		mockStorage.AssertExpectations(t)
	})

	t.Run("Success", func(t *testing.T) {
		mockStorage := new(mocks.MockStorage)

		cartId := 1
		item := models.CartItem{
			Id:       2,
			CartId:   cartId,
			Product:  "item",
			Quantity: 3,
		}
		mustReturn := models.Cart{Id: cartId, Items: []models.CartItem{item}}

		mockStorage.On("ViewCart", mock.Anything, cartId).Return(mustReturn, nil)
		svc := newTestService(mockStorage)

		got, err := svc.ViewCart(context.TODO(), cartId)
		assert.Nil(t, err)
		assert.Equal(t, mustReturn, got)

		mockStorage.AssertExpectations(t)
	})
}
