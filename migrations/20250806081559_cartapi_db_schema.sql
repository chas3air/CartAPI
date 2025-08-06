-- +goose Up
-- +goose StatementBegin
CREATE TABLE cart (
    id SERIAL PRIMARY KEY
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE item (
    id SERIAL PRIMARY KEY,
    product VARCHAR(50),
    quantity INT
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE cart_item (
    id SERIAL PRIMARY KEY,
    cart_id INT,
    item_id INT,
    FOREIGN KEY (cart_id) REFERENCES cart(id) ON DELETE CASCADE,
    FOREIGN KEY (item_id) REFERENCES item(id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE cart_item;
DROP TABLE item;
DROP TABLE cart;
-- +goose StatementEnd
