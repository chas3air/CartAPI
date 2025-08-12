package routes

import (
	carthandler "cartapi/internal/handlers/cart"
	"net/http"
	"strings"
)

type Routes struct {
	cartItemHandler *carthandler.Handler
}

func New(cartItemHandler *carthandler.Handler) *Routes {
	return &Routes{
		cartItemHandler: cartItemHandler,
	}
}

func (r *Routes) Register() {
	// POST /carts
	http.HandleFunc("/carts", r.cartItemHandler.CreateCart)
	http.HandleFunc("/carts/", r.pathParser)
}

func (r *Routes) pathParser(ww http.ResponseWriter, req *http.Request) {
	path := strings.Trim(req.URL.Path, "/")
	parts := strings.Split(path, "/")

	switch {
	case len(parts) == 2 && req.Method == http.MethodGet:
		// GET /carts/{cartId}
		r.cartItemHandler.ViewCart(ww, req, parts[1])
	case len(parts) == 3 && parts[2] == "items" && req.Method == http.MethodPost:
		// POST /carts/{cartId}/items
		r.cartItemHandler.AddToCart(ww, req, parts[1])
	case len(parts) == 4 && parts[2] == "items" && req.Method == http.MethodDelete:
		// DELETE /carts/{cartId}/items/{itemId}
		r.cartItemHandler.RemoveFromCart(ww, req, parts[1], parts[3])
	default:
		http.NotFound(ww, req)
	}

}
