package carthandler

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
		handleServiceError(w, log, err, "Failed to create cart")
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(cart); err != nil {
		log.Error("Failed to respond user", sl.Err(err))
		http.Error(w, "Failed to respond user", http.StatusInternalServerError)
		return
	}
}

// POST /carts/{cartId}/items
func (h *Handler) AddToCart(w http.ResponseWriter, r *http.Request, cartIdStr string) {
	const op = "handlers.cart.AddToCart"
	log := h.log.With("op", op)

	cartId, err := parseCartID(cartIdStr)
	if err != nil {
		log.Error("Invalid cartId parameter", sl.Err(err))
		http.Error(w, "Invalid cart ID", http.StatusBadRequest)
		return
	}

	requestBody, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Error("Cannot read request body", sl.Err(err))
		http.Error(w, "Cannot read request body", http.StatusBadRequest)
		return
	}

	var item models.CartItem
	if err := json.Unmarshal(requestBody, &item); err != nil {
		log.Error("Cannot unmarshal request body", sl.Err(err))
		http.Error(w, "Cannot unmarshal request body", http.StatusBadRequest)
		return
	}

	if item.Product == "" {
		log.Error("Product field is required", sl.Err(errors.New("product field is required")))
		http.Error(w, "Product field is required", http.StatusBadRequest)
		return
	}

	if item.Quantity <= 0 {
		log.Error("Quantity must be greater than zero", sl.Err(errors.New("quantity must be greater than zero")))
		http.Error(w, "Quantity must be greater than zero", http.StatusBadRequest)
		return
	}

	insertedItem, err := h.service.AddToCart(r.Context(), cartId, item)
	if err != nil {
		handleServiceError(w, log, err, "Failed to add to cart")
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(insertedItem); err != nil {
		log.Error("Failed to respond user", sl.Err(err))
		http.Error(w, "Failed to respond user", http.StatusInternalServerError)
		return
	}
}

// DELETE /carts/{cartId}/items/{itemId}
func (h *Handler) RemoveFromCart(w http.ResponseWriter, r *http.Request, cartIdStr string, itemIdStr string) {
	const op = "handlers.cart.RemoveFromCart"
	log := h.log.With("op", op)

	cartId, err := parseCartID(cartIdStr)
	if err != nil {
		log.Error("Invalid cartId parameter", sl.Err(err))
		http.Error(w, "Invalid cart ID", http.StatusBadRequest)
		return
	}

	itemId, err := parseItemID(itemIdStr)
	if err != nil {
		log.Error("Invalid itemId parameter", sl.Err(err))
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	err = h.service.RemoveFromCart(r.Context(), cartId, itemId)
	if err != nil {
		handleServiceError(w, log, err, "Failed to remove from cart")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /carts/{cartId}
func (h *Handler) ViewCart(w http.ResponseWriter, r *http.Request, cartIdStr string) {
	const op = "handlers.cart.ViewCart"
	log := h.log.With("op", op)

	cartId, err := parseCartID(cartIdStr)
	if err != nil {
		log.Error("Invalid cartId parameter", sl.Err(err))
		http.Error(w, "Invalid cart ID", http.StatusBadRequest)
		return
	}

	cart, err := h.service.ViewCart(r.Context(), cartId)
	if err != nil {
		handleServiceError(w, log, err, "Failed to view the cart")
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(cart); err != nil {
		log.Error("Failed to respond user", sl.Err(err))
		http.Error(w, "Failed to respond user", http.StatusInternalServerError)
		return
	}
}

func handleServiceError(w http.ResponseWriter, log *slog.Logger, err error, msg string) {
	if errors.Is(err, serviceerrors.ErrContextCanceled) {
		log.Warn("Context canceled", sl.Err(serviceerrors.ErrContextCanceled))
		http.Error(w, "Context canceled", StatusClientClosedRequest)
	} else if errors.Is(err, serviceerrors.ErrDeadlineExceeded) {
		log.Warn("Deadline exceeded", sl.Err(serviceerrors.ErrDeadlineExceeded))
		http.Error(w, "Deadline exceeded", http.StatusGatewayTimeout)
	} else if errors.Is(err, serviceerrors.ErrNotFound) {
		log.Warn("Cart not found", sl.Err(serviceerrors.ErrNotFound))
		http.Error(w, "Cart not found", http.StatusNotFound)
	} else {
		log.Error(msg, sl.Err(err))
		http.Error(w, msg, http.StatusInternalServerError)
	}
}

func parseCartID(cartIdStr string) (int, error) {
	id, err := strconv.Atoi(cartIdStr)
	if err != nil {
		return 0, errors.New("invalid cartId, must be a positive integer")
	}
	if id <= 0 {
		return 0, errors.New("invalid cartId, must be a positive integer")
	}
	return id, nil
}

func parseItemID(itemIdStr string) (int, error) {
	id, err := strconv.Atoi(itemIdStr)
	if err != nil {
		return 0, errors.New("invalid itemId, must be a positive integer")
	}
	if id <= 0 {
		return 0, errors.New("invalid itemId, must be a positive integer")
	}
	return id, nil
}
