package models

type Cart struct {
	Id    int        `json:"id"`
	Items []CartItem `json:"items"`
}

type CartItem struct {
	Id       int    `json:"id"`
	CartId   int    `json:"cart_id"`
	Product  string `json:"product"`
	Quantity int    `json:"quantity"`
}
