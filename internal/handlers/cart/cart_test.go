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
	"fmt"
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

		handler.AddToCart(ww, req)
		assert.Equal(t, http.StatusBadRequest, ww.Result().StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("No enough parameters", func(t *testing.T) {
		mockService := new(mocks.Service)

		item := models.CartItem{Quantity: 100}
		bItem, _ := json.Marshal(item)

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodPost, "/carts/1/items", bytes.NewBuffer(bItem))
		ww := httptest.NewRecorder()

		handler.AddToCart(ww, req)
		resp := ww.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Wrong parameters", func(t *testing.T) {
		mockService := new(mocks.Service)

		item := models.CartItem{Product: "", Quantity: 0}
		bItem, _ := json.Marshal(item)

		handler := newTestHandler(nil)

		req := httptest.NewRequest(http.MethodPost, "/carts/1/items", bytes.NewBuffer(bItem))
		ww := httptest.NewRecorder()

		handler.AddToCart(ww, req)
		resp := ww.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Context canceled", func(t *testing.T) {
		mockService := new(mocks.Service)
		item := models.CartItem{Id: 1, CartId: 1, Product: "item", Quantity: 100}
		mockService.On("AddToCart", mock.Anything, 1, item).Return(models.CartItem{}, serviceerrors.ErrContextCanceled)

		handler := newTestHandler(mockService)

		byteItem, _ := json.Marshal(item)

		req := httptest.NewRequest(http.MethodPost, "/carts/1/items", bytes.NewBuffer(byteItem))
		ww := httptest.NewRecorder()

		handler.AddToCart(ww, req)
		resp := ww.Result()

		assert.Equal(t, carthandler.StatusClientClosedRequest, resp.StatusCode)
	})

	t.Run("Dedaline exceeded", func(t *testing.T) {
		mockService := new(mocks.Service)
		item := models.CartItem{Id: 1, CartId: 1, Product: "item", Quantity: 100}
		mockService.On("AddToCart", mock.Anything, 1, item).Return(models.CartItem{}, serviceerrors.ErrDeadlineExceeded)

		handler := newTestHandler(mockService)

		byteItem, _ := json.Marshal(item)

		req := httptest.NewRequest(http.MethodPost, "/carts/1/items", bytes.NewBuffer(byteItem))
		ww := httptest.NewRecorder()

		handler.AddToCart(ww, req)
		resp := ww.Result()

		assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)
	})

	t.Run("Failed to create cart", func(t *testing.T) {
		mockService := new(mocks.Service)
		item := models.CartItem{Id: 1, CartId: 1, Product: "item", Quantity: 100}
		mockService.On("AddToCart", mock.Anything, 1, item).Return(models.CartItem{}, errors.New("errors"))

		handler := newTestHandler(mockService)

		byteItem, _ := json.Marshal(item)

		req := httptest.NewRequest(http.MethodPost, "/carts/1/items", bytes.NewBuffer(byteItem))
		ww := httptest.NewRecorder()

		handler.AddToCart(ww, req)
		resp := ww.Result()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("Success", func(t *testing.T) {
		mockService := new(mocks.Service)
		item := models.CartItem{Product: "item", Quantity: 5}
		bItem, _ := json.Marshal(item)

		cartId := 1
		mockService.On("AddToCart", mock.Anything, cartId, item).Return(
			models.CartItem{
				Id:       1,
				CartId:   cartId,
				Product:  item.Product,
				Quantity: item.Quantity,
			}, nil,
		)

		handler := newTestHandler(mockService)
		req := httptest.NewRequest(
			http.MethodPost,
			fmt.Sprintf("/carts/%d/items", cartId),
			bytes.NewBuffer(bItem),
		)
		ww := httptest.NewRecorder()

		handler.AddToCart(ww, req)
		assert.Equal(t, http.StatusCreated, ww.Result().StatusCode)

		mockService.AssertExpectations(t)
	})
}

func TestHandler_RemoveFromCart(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockService := new(mocks.Service)
		mockService.On("RemoveFromCart", mock.Anything, 1, 2).Return(nil)

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodDelete, "/carts/1/items/2", nil)
		ww := httptest.NewRecorder()

		handler.RemoveFromCart(ww, req)
		assert.Equal(t, http.StatusNoContent, ww.Result().StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Invalid arguments", func(t *testing.T) {
		mockService := new(mocks.Service)

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodDelete, "/carts/y", nil)
		ww := httptest.NewRecorder()

		handler.RemoveFromCart(ww, req)
		assert.Equal(t, http.StatusBadRequest, ww.Result().StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Context canceled", func(t *testing.T) {
		mockService := new(mocks.Service)
		mockService.On("RemoveFromCart", mock.Anything, 1, 2).Return(serviceerrors.ErrContextCanceled)

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodDelete, "/carts/1/items/2", nil)
		ww := httptest.NewRecorder()

		handler.RemoveFromCart(ww, req)
		assert.Equal(t, carthandler.StatusClientClosedRequest, ww.Result().StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Deadline exceeded", func(t *testing.T) {
		mockService := new(mocks.Service)
		mockService.On("RemoveFromCart", mock.Anything, 1, 2).Return(serviceerrors.ErrDeadlineExceeded)

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodDelete, "/carts/1/items/2", nil)
		ww := httptest.NewRecorder()

		handler.RemoveFromCart(ww, req)
		assert.Equal(t, http.StatusGatewayTimeout, ww.Result().StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Not found", func(t *testing.T) {
		mockService := new(mocks.Service)
		mockService.On("RemoveFromCart", mock.Anything, 1, 2).Return(serviceerrors.ErrNotFound)

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodDelete, "/carts/1/items/2", nil)
		ww := httptest.NewRecorder()

		handler.RemoveFromCart(ww, req)
		assert.Equal(t, http.StatusNotFound, ww.Result().StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Failed to remove from cart", func(t *testing.T) {
		mockService := new(mocks.Service)
		mockService.On("RemoveFromCart", mock.Anything, 1, 2).Return(errors.New("error"))

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodDelete, "/carts/1/items/2", nil)
		ww := httptest.NewRecorder()

		handler.RemoveFromCart(ww, req)
		assert.Equal(t, http.StatusInternalServerError, ww.Result().StatusCode)

		mockService.AssertExpectations(t)
	})
}

func TestHandler_ViewCart(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockService := new(mocks.Service)
		cartId := 1
		mockService.On("ViewCart", mock.Anything, cartId).Return(models.Cart{
			Id:    cartId,
			Items: []models.CartItem{},
		}, nil)

		handler := newTestHandler(mockService)
		req := httptest.NewRequest(http.MethodGet,
			fmt.Sprintf("/carts/%d", cartId), nil)
		ww := httptest.NewRecorder()

		handler.ViewCart(ww, req)
		assert.Equal(t, http.StatusOK, ww.Result().StatusCode)
		mockService.AssertExpectations(t)
	})

	t.Run("Invalid arguments", func(t *testing.T) {
		mockService := new(mocks.Service)

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodDelete, "/carts/qaqaq", nil)
		ww := httptest.NewRecorder()

		handler.ViewCart(ww, req)
		assert.Equal(t, http.StatusBadRequest, ww.Result().StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Context canceled", func(t *testing.T) {
		mockService := new(mocks.Service)
		mockService.On("ViewCart", mock.Anything, 1).Return(models.Cart{}, serviceerrors.ErrContextCanceled)

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodDelete, "/carts/1", nil)
		ww := httptest.NewRecorder()

		handler.ViewCart(ww, req)
		assert.Equal(t, carthandler.StatusClientClosedRequest, ww.Result().StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Deadline exceeded", func(t *testing.T) {
		mockService := new(mocks.Service)
		mockService.On("ViewCart", mock.Anything, 1).Return(models.Cart{}, serviceerrors.ErrDeadlineExceeded)

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodDelete, "/carts/1", nil)
		ww := httptest.NewRecorder()

		handler.ViewCart(ww, req)
		assert.Equal(t, http.StatusGatewayTimeout, ww.Result().StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Not found", func(t *testing.T) {
		mockService := new(mocks.Service)
		mockService.On("ViewCart", mock.Anything, 1).Return(models.Cart{}, serviceerrors.ErrNotFound)

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodDelete, "/carts/1", nil)
		ww := httptest.NewRecorder()

		handler.ViewCart(ww, req)
		assert.Equal(t, http.StatusNotFound, ww.Result().StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Failed to view the cart", func(t *testing.T) {
		mockService := new(mocks.Service)
		mockService.On("ViewCart", mock.Anything, 1).Return(models.Cart{}, errors.New("error"))

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodDelete, "/carts/1", nil)
		ww := httptest.NewRecorder()

		handler.ViewCart(ww, req)
		assert.Equal(t, http.StatusInternalServerError, ww.Result().StatusCode)

		mockService.AssertExpectations(t)
	})
}
