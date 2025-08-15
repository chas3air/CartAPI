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
	tests := []struct {
		name         string
		setupMock    func(s *mocks.Service)
		reqContext   context.Context
		expectedCode int
		checkBody    bool
	}{
		{
			name: "Success",
			setupMock: func(s *mocks.Service) {
				s.On("CreateCart", mock.Anything).Return(models.Cart{Id: 1, Items: []models.CartItem{}}, nil)
			},
			reqContext:   context.Background(),
			expectedCode: http.StatusCreated,
			checkBody:    true,
		},
		{
			name: "Context canceled",
			setupMock: func(s *mocks.Service) {
				s.On("CreateCart", mock.Anything).Return(models.Cart{}, serviceerrors.ErrContextCanceled)
			},
			reqContext:   func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
			expectedCode: carthandler.StatusClientClosedRequest,
		},
		{
			name: "Deadline exceeded",
			setupMock: func(s *mocks.Service) {
				s.On("CreateCart", mock.Anything).Return(models.Cart{}, serviceerrors.ErrDeadlineExceeded)
			},
			reqContext:   func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
			expectedCode: http.StatusGatewayTimeout,
		},
		{
			name: "Failed to create cart",
			setupMock: func(s *mocks.Service) {
				s.On("CreateCart", mock.Anything).Return(models.Cart{}, errors.New("error"))
			},
			reqContext:   context.Background(),
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mocks.Service)
			tt.setupMock(mockService)

			handler := newTestHandler(mockService)
			req := httptest.NewRequest(http.MethodPost, "/carts", nil).WithContext(tt.reqContext)
			ww := httptest.NewRecorder()

			handler.CreateCart(ww, req)
			resp := ww.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			if tt.checkBody {
				var got models.Cart
				err := json.NewDecoder(resp.Body).Decode(&got)
				assert.NoError(t, err)
				assert.Equal(t, 1, got.Id)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_AddToCart(t *testing.T) {
	tests := []struct {
		name         string
		cartId       string
		body         []byte
		setupMock    func(s *mocks.Service)
		expectedCode int
		checkBody    bool
	}{
		{
			name:         "Empty body",
			cartId:       "1",
			body:         nil,
			setupMock:    func(s *mocks.Service) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Invalid cartId",
			cartId:       "abc",
			body:         []byte(`{"product":"test product","quantity":1}`),
			setupMock:    func(s *mocks.Service) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Invalid JSON",
			cartId:       "1",
			body:         []byte("{invalid json"),
			setupMock:    func(s *mocks.Service) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Validation failed",
			cartId:       "1",
			body:         []byte(`{"product":""}`),
			setupMock:    func(s *mocks.Service) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "Success",
			cartId: "1",
			setupMock: func(s *mocks.Service) {
				item := models.CartItem{Product: "item", Quantity: 5}
				returnItem := models.CartItem{Id: 1, CartId: 1, Product: item.Product, Quantity: item.Quantity}
				s.On("AddToCart", mock.Anything, 1, item).Return(returnItem, nil)
			},
			body:         []byte(`{"product":"item","quantity":5}`),
			expectedCode: http.StatusCreated,
			checkBody:    true,
		},
		{
			name:   "Service error",
			cartId: "1",
			setupMock: func(s *mocks.Service) {
				item := models.CartItem{Product: "item", Quantity: 5}
				s.On("AddToCart", mock.Anything, 1, item).Return(models.CartItem{}, errors.New("service failure"))
			},
			body:         []byte(`{"product":"item","quantity":5}`),
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mocks.Service)
			tt.setupMock(mockService)
			handler := newTestHandler(mockService)

			req := httptest.NewRequest(http.MethodPost, "/carts/"+tt.cartId+"/items", bytes.NewBuffer(tt.body))
			ww := httptest.NewRecorder()

			handler.AddToCart(ww, req, tt.cartId)
			resp := ww.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			if tt.checkBody && resp.StatusCode == http.StatusCreated {
				var got models.CartItem
				err := json.NewDecoder(resp.Body).Decode(&got)
				assert.NoError(t, err)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_RemoveFromCart(t *testing.T) {
	tests := []struct {
		name         string
		cartId       string
		itemId       string
		setupMock    func(s *mocks.Service)
		expectedCode int
	}{
		{
			name:   "Success",
			cartId: "1",
			itemId: "2",
			setupMock: func(s *mocks.Service) {
				s.On("RemoveFromCart", mock.Anything, 1, 2).Return(nil)
			},
			expectedCode: http.StatusNoContent,
		},
		{
			name:         "Invalid cartId and itemId",
			cartId:       "a",
			itemId:       "b",
			setupMock:    func(s *mocks.Service) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "Service error",
			cartId: "1",
			itemId: "2",
			setupMock: func(s *mocks.Service) {
				s.On("RemoveFromCart", mock.Anything, 1, 2).Return(errors.New("remove error"))
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mocks.Service)
			tt.setupMock(mockService)
			handler := newTestHandler(mockService)

			req := httptest.NewRequest(http.MethodDelete, "/carts/"+tt.cartId+"/items/"+tt.itemId, nil)
			ww := httptest.NewRecorder()

			handler.RemoveFromCart(ww, req, tt.cartId, tt.itemId)
			resp := ww.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_ViewCart(t *testing.T) {
	tests := []struct {
		name         string
		cartId       string
		setupMock    func(s *mocks.Service)
		expectedCode int
		checkBody    bool
	}{
		{
			name:   "Success",
			cartId: "1",
			setupMock: func(s *mocks.Service) {
				s.On("ViewCart", mock.Anything, 1).Return(models.Cart{Id: 1, Items: []models.CartItem{}}, nil)
			},
			expectedCode: http.StatusOK,
			checkBody:    true,
		},
		{
			name:         "Invalid cartId",
			cartId:       "abc",
			setupMock:    func(s *mocks.Service) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "Not found error",
			cartId: "1",
			setupMock: func(s *mocks.Service) {
				s.On("ViewCart", mock.Anything, 1).Return(models.Cart{}, serviceerrors.ErrNotFound)
			},
			expectedCode: http.StatusNotFound,
		},
		{
			name:   "Service error",
			cartId: "1",
			setupMock: func(s *mocks.Service) {
				s.On("ViewCart", mock.Anything, 1).Return(models.Cart{}, errors.New("service error"))
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mocks.Service)
			tt.setupMock(mockService)
			handler := newTestHandler(mockService)

			req := httptest.NewRequest(http.MethodGet, "/carts/"+tt.cartId, nil)
			ww := httptest.NewRecorder()

			handler.ViewCart(ww, req, tt.cartId)
			resp := ww.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			if tt.checkBody && resp.StatusCode == http.StatusOK {
				var got models.Cart
				err := json.NewDecoder(resp.Body).Decode(&got)
				assert.NoError(t, err)
			}

			mockService.AssertExpectations(t)
		})
	}
}
