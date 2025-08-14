package carthandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	carthandler "cartapi/internal/handlers/cart"
	"cartapi/internal/handlers/cart/mocks"
	"cartapi/internal/models"
	serviceerrors "cartapi/internal/service"
	"cartapi/pkg/lib/logger/slogdiscard"

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
		defer resp.Body.Close()

		assert.Equal(t, carthandler.StatusClientClosedRequest, resp.StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Deadline exceeded", func(t *testing.T) {
		mockService := new(mocks.Service)
		mockService.On("CreateCart", mock.Anything).Return(models.Cart{}, serviceerrors.ErrDeadlineExceeded)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		handler := newTestHandler(mockService)

		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/carts", nil)
		ww := httptest.NewRecorder()

		handler.CreateCart(ww, req)
		resp := ww.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Failed to create cart", func(t *testing.T) {
		mockService := new(mocks.Service)
		mockService.On("CreateCart", mock.Anything).Return(models.Cart{}, errors.New("error"))

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodPost, "/carts", nil)
		ww := httptest.NewRecorder()

		handler.CreateCart(ww, req)
		resp := ww.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		mockService.AssertExpectations(t)
	})
}

func TestHandler_AddToCart(t *testing.T) {
	t.Run("Empty body", func(t *testing.T) {
		mockService := new(mocks.Service)

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodPost, "/carts/1/items", nil)
		ww := httptest.NewRecorder()

		handler.AddToCart(ww, req, "1")

		assert.Equal(t, http.StatusBadRequest, ww.Result().StatusCode)
		mockService.AssertExpectations(t)
	})

	t.Run("Invalid cartId", func(t *testing.T) {
		mockService := new(mocks.Service)

		handler := newTestHandler(mockService)

		item := models.CartItem{
			Product:  "test product",
			Quantity: 1,
		}
		bItem, _ := json.Marshal(item)

		req := httptest.NewRequest(http.MethodPost, "/carts/abc/items", bytes.NewBuffer(bItem))
		ww := httptest.NewRecorder()

		handler.AddToCart(ww, req, "abc")

		assert.Equal(t, http.StatusBadRequest, ww.Result().StatusCode)
		mockService.AssertExpectations(t)
	})

	t.Run("Invalid JSON body", func(t *testing.T) {
		mockService := new(mocks.Service)

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodPost, "/carts/1/items", bytes.NewBuffer([]byte("{invalid json")))
		ww := httptest.NewRecorder()

		handler.AddToCart(ww, req, "1")

		assert.Equal(t, http.StatusBadRequest, ww.Result().StatusCode)
		mockService.AssertExpectations(t)
	})

	t.Run("Validation failed", func(t *testing.T) {
		mockService := new(mocks.Service)

		handler := newTestHandler(mockService)

		item := models.CartItem{
			Product: "",
		}
		bItem, _ := json.Marshal(item)

		req := httptest.NewRequest(http.MethodPost, "/carts/1/items", bytes.NewBuffer(bItem))
		ww := httptest.NewRecorder()

		handler.AddToCart(ww, req, "1")

		assert.Equal(t, http.StatusBadRequest, ww.Result().StatusCode)
		mockService.AssertExpectations(t)
	})

	t.Run("Success", func(t *testing.T) {
		mockService := new(mocks.Service)

		cartId := "1"
		cartIdInt := 1
		item := models.CartItem{Product: "item", Quantity: 5}
		returnItem := models.CartItem{
			Id:       1,
			CartId:   cartIdInt,
			Product:  item.Product,
			Quantity: item.Quantity,
		}

		mockService.On("AddToCart", mock.Anything, cartIdInt, item).Return(returnItem, nil)

		handler := newTestHandler(mockService)
		bItem, _ := json.Marshal(item)
		req := httptest.NewRequest(http.MethodPost, "/carts/"+cartId+"/items", bytes.NewBuffer(bItem))
		ww := httptest.NewRecorder()

		handler.AddToCart(ww, req, cartId)

		resp := ww.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var got models.CartItem
		err := json.NewDecoder(resp.Body).Decode(&got)
		assert.NoError(t, err)
		assert.Equal(t, returnItem, got)

		mockService.AssertExpectations(t)
	})

	t.Run("Service error", func(t *testing.T) {
		mockService := new(mocks.Service)

		cartId := "1"
		cartIdInt := 1
		item := models.CartItem{Product: "item", Quantity: 5}

		mockService.On("AddToCart", mock.Anything, cartIdInt, item).Return(models.CartItem{}, errors.New("service failure"))

		handler := newTestHandler(mockService)
		bItem, _ := json.Marshal(item)
		req := httptest.NewRequest(http.MethodPost, "/carts/"+cartId+"/items", bytes.NewBuffer(bItem))
		ww := httptest.NewRecorder()

		handler.AddToCart(ww, req, cartId)

		assert.Equal(t, http.StatusInternalServerError, ww.Result().StatusCode)
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

		handler.RemoveFromCart(ww, req, "1", "2")

		assert.Equal(t, http.StatusNoContent, ww.Result().StatusCode)
		mockService.AssertExpectations(t)
	})

	t.Run("Invalid cartId and itemId", func(t *testing.T) {
		mockService := new(mocks.Service)

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodDelete, "/carts/a/items/b", nil)
		ww := httptest.NewRecorder()

		handler.RemoveFromCart(ww, req, "a", "b")

		assert.Equal(t, http.StatusBadRequest, ww.Result().StatusCode)
		mockService.AssertExpectations(t)
	})

	t.Run("Service error", func(t *testing.T) {
		mockService := new(mocks.Service)
		mockService.On("RemoveFromCart", mock.Anything, 1, 2).Return(errors.New("remove error"))

		handler := newTestHandler(mockService)

		req := httptest.NewRequest(http.MethodDelete, "/carts/1/items/2", nil)
		ww := httptest.NewRecorder()

		handler.RemoveFromCart(ww, req, "1", "2")

		assert.Equal(t, http.StatusInternalServerError, ww.Result().StatusCode)

		mockService.AssertExpectations(t)
	})
}

func TestHandler_ViewCart(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockService := new(mocks.Service)
		cartIdInt := 1
		cart := models.Cart{
			Id:    cartIdInt,
			Items: []models.CartItem{},
		}
		mockService.On("ViewCart", mock.Anything, cartIdInt).Return(cart, nil)

		handler := newTestHandler(mockService)
		req := httptest.NewRequest(http.MethodGet, "/carts/1", nil)
		ww := httptest.NewRecorder()

		handler.ViewCart(ww, req, "1")

		assert.Equal(t, http.StatusOK, ww.Result().StatusCode)

		var got models.Cart
		err := json.NewDecoder(ww.Result().Body).Decode(&got)
		assert.NoError(t, err)
		assert.Equal(t, cart, got)

		mockService.AssertExpectations(t)
	})

	t.Run("Invalid cartId", func(t *testing.T) {
		mockService := new(mocks.Service)

		handler := newTestHandler(mockService)
		req := httptest.NewRequest(http.MethodGet, "/carts/abc", nil)
		ww := httptest.NewRecorder()

		handler.ViewCart(ww, req, "abc")

		assert.Equal(t, http.StatusBadRequest, ww.Result().StatusCode)
		mockService.AssertExpectations(t)
	})

	t.Run("Not found error", func(t *testing.T) {
		mockService := new(mocks.Service)
		cartIdInt := 1
		mockService.On("ViewCart", mock.Anything, cartIdInt).Return(models.Cart{}, serviceerrors.ErrNotFound)

		handler := newTestHandler(mockService)
		req := httptest.NewRequest(http.MethodGet, "/carts/1", nil)
		ww := httptest.NewRecorder()

		handler.ViewCart(ww, req, "1")

		assert.Equal(t, http.StatusNotFound, ww.Result().StatusCode)

		mockService.AssertExpectations(t)
	})

	t.Run("Service error", func(t *testing.T) {
		mockService := new(mocks.Service)
		cartIdInt := 1
		mockService.On("ViewCart", mock.Anything, cartIdInt).Return(models.Cart{}, errors.New("service error"))

		handler := newTestHandler(mockService)
		req := httptest.NewRequest(http.MethodGet, "/carts/1", nil)
		ww := httptest.NewRecorder()

		handler.ViewCart(ww, req, "1")

		assert.Equal(t, http.StatusInternalServerError, ww.Result().StatusCode)

		mockService.AssertExpectations(t)
	})
}
