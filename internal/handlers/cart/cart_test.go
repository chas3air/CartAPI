package carthandler_test

import (
	"bytes"
	carthandler "cartapi/internal/handlers/cart"
	"cartapi/internal/handlers/cart/mocks"
	"cartapi/internal/models"
	serviceerrors "cartapi/internal/service"
	"cartapi/pkg/lib/logger/slogdiscard"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestHandler(service *mocks.Service) *carthandler.Handler {
	logger := slogdiscard.NewDiscardLogger()
	return carthandler.New(logger, service)
}

func TestHandler_CreateCart(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockService := new(mocks.Service)

		cart := models.Cart{
			Id:    1,
			Items: []models.CartItem{},
		}

		mockService.On("CreateCart", mock.Anything).Return(cart, nil)

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodPost, "/carts", nil)
		ww := httptest.NewRecorder()

		handler.CreateCart(ww, req)
		resp := ww.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var got models.Cart
		err := json.NewDecoder(resp.Body).Decode(&got)
		assert.NoError(t, err)
		assert.Equal(t, cart, got)

		mockService.AssertExpectations(t)
	})

	t.Run("Context canceled", func(t *testing.T) {
		mockService := new(mocks.Service)
		mockService.On("CreateCart", mock.Anything).Return(models.Cart{}, serviceerrors.ErrContextCanceled)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		handler := newTestHandler(mockService)

		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/carts", nil)

		ww := httptest.NewRecorder()

		handler.CreateCart(ww, req)
		resp := ww.Result()

		assert.Equal(t, carthandler.StatusClientClosedRequest, resp.StatusCode)
	})

	t.Run("Dedaline exceeded", func(t *testing.T) {
		mockService := new(mocks.Service)
		mockService.On("CreateCart", mock.Anything).Return(models.Cart{}, serviceerrors.ErrDeadlineExceeded)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		handler := newTestHandler(mockService)

		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/carts", nil)

		ww := httptest.NewRecorder()

		handler.CreateCart(ww, req)
		resp := ww.Result()

		assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)
	})

	t.Run("Failed to create cart", func(t *testing.T) {
		mockService := new(mocks.Service)
		mockService.On("CreateCart", mock.Anything).Return(models.Cart{}, errors.New("error"))

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		handler := newTestHandler(mockService)

		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/carts", nil)

		ww := httptest.NewRecorder()

		handler.CreateCart(ww, req)
		resp := ww.Result()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

func TestHandler_AddToCart(t *testing.T) {
	t.Run("Empty body", func(t *testing.T) {
		mockService := new(mocks.Service)

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodPost, "/carts/1/items", nil)
		ww := httptest.NewRecorder()

		cartId := "1"
		handler.AddToCart(ww, req, cartId)

		assert.Equal(t, http.StatusBadRequest, ww.Result().StatusCode)
		mockService.AssertExpectations(t)
	})

	t.Run("Success", func(t *testing.T) {
		mockService := new(mocks.Service)
		item := models.CartItem{Product: "item", Quantity: 5}
		bItem, _ := json.Marshal(item)

		cartId := "1"
		cartIdInt := 1

		mockService.On("AddToCart", mock.Anything, cartIdInt, item).Return(
			models.CartItem{
				Id:       1,
				CartId:   cartIdInt,
				Product:  item.Product,
				Quantity: item.Quantity,
			}, nil,
		)

		handler := newTestHandler(mockService)
		req := httptest.NewRequest(
			http.MethodPost,
			"/carts/"+cartId+"/items",
			bytes.NewBuffer(bItem),
		)
		ww := httptest.NewRecorder()

		handler.AddToCart(ww, req, cartId)
		assert.Equal(t, http.StatusCreated, ww.Result().StatusCode)

		mockService.AssertExpectations(t)
	})

	// остальные под-тесты аналогично — вызывайте AddToCart с cartId из extractCartIdFromPath
}

func TestHandler_RemoveFromCart(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockService := new(mocks.Service)
		mockService.On("RemoveFromCart", mock.Anything, 1, 2).Return(nil)

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodDelete, "/carts/1/items/2", nil)
		ww := httptest.NewRecorder()

		cartId := "1"
		itemId := "2"
		handler.RemoveFromCart(ww, req, cartId, itemId)

		assert.Equal(t, http.StatusNoContent, ww.Result().StatusCode)
		mockService.AssertExpectations(t)
	})

	t.Run("Invalid arguments", func(t *testing.T) {
		mockService := new(mocks.Service)

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodDelete, "/carts/y/items/z", nil)
		ww := httptest.NewRecorder()

		cartId := "y"
		itemId := "z"
		handler.RemoveFromCart(ww, req, cartId, itemId)

		assert.Equal(t, http.StatusBadRequest, ww.Result().StatusCode)
		mockService.AssertExpectations(t)
	})

	// остальные под-тесты аналогично
}

func TestHandler_ViewCart(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockService := new(mocks.Service)
		cartIdInt := 1
		mockService.On("ViewCart", mock.Anything, cartIdInt).Return(models.Cart{
			Id:    cartIdInt,
			Items: []models.CartItem{},
		}, nil)

		handler := newTestHandler(mockService)
		req := httptest.NewRequest(http.MethodGet, "/carts/1", nil)
		ww := httptest.NewRecorder()

		cartId := "1"
		handler.ViewCart(ww, req, cartId)

		assert.Equal(t, http.StatusOK, ww.Result().StatusCode)
		mockService.AssertExpectations(t)
	})

	t.Run("Invalid arguments", func(t *testing.T) {
		mockService := new(mocks.Service)

		handler := newTestHandler(mockService)
		req := httptest.NewRequest(http.MethodGet, "/carts/qaqaq", nil)
		ww := httptest.NewRecorder()

		cartId := "qaqaq"
		handler.ViewCart(ww, req, cartId)

		assert.Equal(t, http.StatusBadRequest, ww.Result().StatusCode)
		mockService.AssertExpectations(t)
	})

	// остальные под-тесты — аналогично
}
