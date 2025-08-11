package urlparser

import (
	"errors"
	"strconv"
	"strings"
)

type PathParams struct {
	CartId int
	ItemId int
}

func ParseCartPath(path string) (PathParams, error) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")

	params := PathParams{}

	switch len(parts) {
	case 2:
		if parts[0] != "carts" {
			return params, errors.New("invalid path, expected /carts/{cartId}")
		}
		cartId, err := strconv.Atoi(parts[1])
		if err != nil {
			return params, errors.New("invalid cartId, must be int")
		}
		params.CartId = cartId
		return params, nil
	case 3:
		if parts[0] != "carts" || parts[2] != "items" {
			return params, errors.New("invalid path, expected /carts/{cartId}/items")
		}
		cartId, err := strconv.Atoi(parts[1])
		if err != nil {
			return params, errors.New("invalid cartId, must be int")
		}
		params.CartId = cartId
		return params, nil
	case 4:
		if parts[0] != "carts" || parts[2] != "items" {
			return params, errors.New("invalid path, expected /carts/{cartId}/items/{itemId}")
		}
		cartId, err := strconv.Atoi(parts[1])
		if err != nil {
			return params, errors.New("invalid cartId, must be int")
		}
		itemId, err := strconv.Atoi(parts[3])
		if err != nil {
			return params, errors.New("invalid itemId, must be int")
		}
		params.CartId = cartId
		params.ItemId = itemId
		return params, nil

	default:
		return params, errors.New("wrong url format")
	}
}
