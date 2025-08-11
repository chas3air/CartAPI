package models

type Cart struct {
	Id    int        `json:"id"`
	Items []CartItem `json:"items"`
}

type CartItem struct {
	Id       int    `json:"id" db:"id"`
	CartId   int    `json:"cart_id" db:"cart_id"`
	Product  string `json:"product" validate:"required" db:"product"`
	Quantity int    `json:"quantity" validate:"required,gt=0" db:"quantity"`
}
