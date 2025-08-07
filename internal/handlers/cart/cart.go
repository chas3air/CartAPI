package cart

import (
	"cartapi/internal/models"
	serviceerrors "cartapi/internal/service"
	"cartapi/pkg/lib/logger/sl"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

const StatusClientClosedRequest = 499

type CartItemService interface {
	CreateCart(ctx context.Context) (models.Cart, error)
	AddToCart(ctx context.Context, cartId int, item models.CartItem) (models.CartItem, error)
	RemoveFromCart(ctx context.Context, cartId int, itemId int) error
	ViewCart(ctx context.Context, cartId int) (models.Cart, error)
}

type Handler struct {
	log     *slog.Logger
	service CartItemService
}

func New(log *slog.Logger, service CartItemService) *Handler {
	return &Handler{
		log:     log,
		service: service,
	}
}

// POST /carts
func (h *Handler) CreateCart(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.cart.CreateCart"
	log := h.log.With("op", op)

	cart, err := h.service.CreateCart(r.Context())
	if err != nil {
		if errors.Is(err, serviceerrors.ErrContextCanceled) {
			log.Warn("Context canceled", sl.Err(serviceerrors.ErrContextCanceled))
			http.Error(w, "Context canceled", StatusClientClosedRequest)
			return
		} else if errors.Is(err, serviceerrors.ErrDeadlineExceeded) {
			log.Warn("Deadline exceeded", sl.Err(serviceerrors.ErrDeadlineExceeded))
			http.Error(w, "Deadline exceeded", http.StatusGatewayTimeout)
			return
		} else {
			log.Error("Failed to create cart", sl.Err(err))
			http.Error(w, "Failed to create cart", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(cart); err != nil {
		log.Error("Failed to responde user", sl.Err(err))
		http.Error(w, "Failed to responde user", http.StatusInternalServerError)
		return
	}
}

// POST /carts/cartId/items
func (h *Handler) AddToCart(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.cart.AddToCart"
	log := h.log.With("op", op)

	urlComponents := strings.Split(r.URL.Path, "/")
	scartId := urlComponents[1]

	cartId, err := strconv.Atoi(scartId)
	if err != nil {
		log.Error("CartID must be int", sl.Err(err))
		http.Error(w, "CartID must be int", http.StatusBadRequest)
		return
	}

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("Cannot read request body", sl.Err(err))
		http.Error(w, "Cannot read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var item models.CartItem
	if err := json.Unmarshal(requestBody, &item); err != nil {
		log.Error("Cannot unmarshal request body", sl.Err(err))
		http.Error(w, "Cannot unmarshal request body", http.StatusBadRequest)
	}

	if err := validator.New().Struct(requestBody); err != nil {
		log.Error("Failed to validate", sl.Err(err))
		http.Error(w, "Failed to validate", http.StatusBadRequest)
		return
	}

	insertedItem, err := h.service.AddToCart(r.Context(), cartId, item)
	if err != nil {
		if errors.Is(err, serviceerrors.ErrContextCanceled) {
			log.Warn("Context canceled", sl.Err(serviceerrors.ErrContextCanceled))
			http.Error(w, "Context canceled", StatusClientClosedRequest)
			return
		} else if errors.Is(err, serviceerrors.ErrDeadlineExceeded) {
			log.Warn("Deadline exceeded", sl.Err(serviceerrors.ErrDeadlineExceeded))
			http.Error(w, "Deadline exceeded", http.StatusGatewayTimeout)
			return
		} else {
			log.Error("Failed to create cart", sl.Err(err))
			http.Error(w, "Failed to create cart", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(insertedItem); err != nil {
		log.Error("Failed to responde user", sl.Err(err))
		http.Error(w, "Failed to responde user", http.StatusInternalServerError)
		return
	}
}

// DELETE /carts/cartId/items/itemId
func (h *Handler) RemoveFromCart(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.cart.RemoveFromCart"
	log := h.log.With("op", op)

	urlComponents := strings.Split(r.URL.Path, "/")
	scartId := urlComponents[2]
	sitemId := urlComponents[4]

	cartId, err := strconv.Atoi(scartId)
	if err != nil {
		log.Error("CartId must be int", sl.Err(err))
		http.Error(w, "CartId must be int", http.StatusBadRequest)
		return
	}

	itemId, err := strconv.Atoi(sitemId)
	if err != nil {
		log.Error("ItemId must be int", sl.Err(err))
		http.Error(w, "ItemId must be int", http.StatusBadRequest)
		return
	}

	err = h.service.RemoveFromCart(r.Context(), cartId, itemId)
	if err != nil {
		if errors.Is(err, serviceerrors.ErrContextCanceled) {
			log.Warn("Context canceled", sl.Err(serviceerrors.ErrContextCanceled))
			http.Error(w, "Context canceled", StatusClientClosedRequest)
			return
		} else if errors.Is(err, serviceerrors.ErrDeadlineExceeded) {
			log.Warn("Deadline exceeded", sl.Err(serviceerrors.ErrDeadlineExceeded))
			http.Error(w, "Deadline exceeded", http.StatusGatewayTimeout)
			return
		} else if errors.Is(err, serviceerrors.ErrNotFound) {
			log.Warn("Cart not found", sl.Err(serviceerrors.ErrNotFound))
			http.NotFound(w, r)
			return
		} else {
			log.Error("Failed to create cart", sl.Err(err))
			http.Error(w, "Failed to create cart", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /carts/cartId
func (h *Handler) ViewCart(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.cart.ViewCart"
	log := h.log.With("op", op)

	urlComponents := strings.Split(r.URL.Path, "/")
	scartId := urlComponents[2]

	cartId, err := strconv.Atoi(scartId)
	if err != nil {
		log.Error("CartId must be int", sl.Err(err))
		http.Error(w, "CartId must be int", http.StatusBadRequest)
		return
	}

	cart, err := h.service.ViewCart(r.Context(), cartId)
	if err != nil {
		if errors.Is(err, serviceerrors.ErrContextCanceled) {
			log.Warn("Context canceled", sl.Err(serviceerrors.ErrContextCanceled))
			http.Error(w, "Context canceled", StatusClientClosedRequest)
			return
		} else if errors.Is(err, serviceerrors.ErrDeadlineExceeded) {
			log.Warn("Deadline exceeded", sl.Err(serviceerrors.ErrDeadlineExceeded))
			http.Error(w, "Deadline exceeded", http.StatusGatewayTimeout)
			return
		} else if errors.Is(err, serviceerrors.ErrNotFound) {
			log.Warn("Cart not found", sl.Err(serviceerrors.ErrNotFound))
			http.NotFound(w, r)
			return
		} else {
			log.Error("Failed to create cart", sl.Err(err))
			http.Error(w, "Failed to create cart", http.StatusInternalServerError)
			return
		}
	}

	if err := json.NewEncoder(w).Encode(cart); err != nil {
		log.Error("Failed to responde user", sl.Err(err))
		http.Error(w, "Failed to responde user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
