-- +goose Up
-- +goose StatementBegin
CREATE TABLE cart (
    id SERIAL PRIMARY KEY
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE item (
    id SERIAL PRIMARY KEY,
    cart_id INT NOT NULL,
    product VARCHAR(50) NOT NULL,
    quantity INT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE item;
DROP TABLE cart;
-- +goose StatementEnd
