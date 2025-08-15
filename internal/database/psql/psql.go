package psql

import (
	databaseerrors "cartapi/internal/database"
	"cartapi/internal/models"
	"cartapi/pkg/lib/logger/sl"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "github.com/lib/pq"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
)

type Storage struct {
	log *slog.Logger
	db  *sqlx.DB
}

func New(log *slog.Logger, connStr string) (*Storage, error) {
	const op = "database.psql.New"
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		log.With("op", op).Error("Error connect to database", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	wd, err := os.Getwd()
	if err != nil {
		log.With("op", op).Error("Error getting work dir", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	migrationsPath := filepath.Join(wd, "migrations")

	if err := goose.Up(db.DB, migrationsPath); err != nil {
		log.With("op", op).Error("Error applying migrations", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{
		log: log,
		db:  db,
	}, nil
}

func NewWithParams(log *slog.Logger, db *sqlx.DB) *Storage {
	return &Storage{
		log: log,
		db:  db,
	}
}

func (s *Storage) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}
	return nil
}

func (s *Storage) CreateCart(ctx context.Context) (models.Cart, error) {
	const op = "database.psql.CreateCart"
	log := s.log.With("op", op)

	select {
	case <-ctx.Done():
		log.Error("Context is over", sl.Err(ctx.Err()))
		return models.Cart{}, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	var cartId int
	err := s.db.QueryRowxContext(ctx, `
        INSERT INTO cart
        DEFAULT VALUES
        RETURNING id;
    `).Scan(&cartId)
	if err != nil {
		log.Error("Error creating cart", sl.Err(err))
		return models.Cart{}, fmt.Errorf("%s: %w", op, err)
	}

	return models.Cart{Id: cartId}, nil
}

func (s *Storage) AddToCart(ctx context.Context, cartId int, item models.CartItem) (models.CartItem, error) {
	const op = "database.psql.AddToCart"
	log := s.log.With("op", op)

	select {
	case <-ctx.Done():
		log.Error("Context is over", sl.Err(ctx.Err()))
		return models.CartItem{}, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	tx, err := s.db.Beginx()
	if err != nil {
		log.Error("Failed to begin transaction", sl.Err(err))
		return models.CartItem{}, fmt.Errorf("%s: %w", op, err)
	}
	defer tx.Rollback()

	var existsChecker int
	if err = tx.QueryRowxContext(ctx, `SELECT id FROM cart WHERE id=$1;`, cartId).Scan(&existsChecker); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn("Cart doesn't exist", sl.Err(databaseerrors.ErrNotFound))
			return models.CartItem{}, fmt.Errorf("%s: %w", op, databaseerrors.ErrNotFound)
		}
		log.Error("Error checking cart existence", sl.Err(err))
		return models.CartItem{}, fmt.Errorf("%s: %w", op, err)
	}

	var itemId int
	row := tx.QueryRowxContext(ctx, `
		INSERT INTO item (cart_id, product, quantity)
		VALUES ($1, $2, $3)
		RETURNING id;
  `, cartId, item.Product, item.Quantity)
	if err := row.Scan(&itemId); err != nil {
		log.Error("Failed to insert item", sl.Err(err))
		return models.CartItem{}, fmt.Errorf("%s: %w", op, err)
	}

	if err := tx.Commit(); err != nil {
		log.Error("Failed to commit transaction", sl.Err(err))
		return models.CartItem{}, fmt.Errorf("%s: %w", op, err)
	}

	return models.CartItem{
		Id:       itemId,
		CartId:   cartId,
		Product:  item.Product,
		Quantity: item.Quantity,
	}, nil
}

func (s *Storage) RemoveFromCart(ctx context.Context, cartId int, itemId int) error {
	const op = "database.psql.RemoveFromCart"
	log := s.log.With("op", op)

	select {
	case <-ctx.Done():
		log.Error("Context is over", sl.Err(ctx.Err()))
		return fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	tx, err := s.db.Beginx()
	if err != nil {
		log.Error("Failed to begin transaction", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}
	defer tx.Rollback()

	var existsChecker int
	if err = tx.QueryRowxContext(ctx, `SELECT id FROM cart WHERE id=$1;`, cartId).Scan(&existsChecker); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn("Cart doesn't exist", sl.Err(databaseerrors.ErrNotFound))
			return fmt.Errorf("%s: %w", op, databaseerrors.ErrNotFound)
		}
		log.Error("Error checking cart existence", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	var itemCartId int
	if err = tx.QueryRowxContext(ctx, `SELECT cart_id FROM item WHERE id=$1;`, itemId).Scan(&itemCartId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn("Cart item doesn't exist", sl.Err(databaseerrors.ErrNotFound))
			return fmt.Errorf("%s: %w", op, databaseerrors.ErrNotFound)
		}
		log.Error("Error checking cart item existence", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM item WHERE id=$1;`, itemId); err != nil {
		log.Error("Failed to delete item", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	if err := tx.Commit(); err != nil {
		log.Error("Failed to commit transaction", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) ViewCart(ctx context.Context, cartId int) (models.Cart, error) {
	const op = "database.psql.ViewCart"
	log := s.log.With("op", op)

	select {
	case <-ctx.Done():
		log.Error("Context is over", sl.Err(ctx.Err()))
		return models.Cart{}, fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	var count int
	row := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM cart WHERE id=$1;
	`, cartId)

	if err := row.Scan(&count); err != nil {
		log.Error("Failed to check cart existence", sl.Err(err))
		return models.Cart{}, fmt.Errorf("%s: %w", op, err)
	}

	if count == 0 {
		log.Warn("Cart doesn't exist", sl.Err(databaseerrors.ErrNotFound))
		return models.Cart{}, fmt.Errorf("%s: %w", op, databaseerrors.ErrNotFound)
	}

	rows, err := s.db.QueryxContext(ctx, `
	SELECT id, cart_id, product, quantity FROM item
	WHERE cart_id=$1;
`, cartId)
	if err != nil {
		log.Error("Failed to query items", sl.Err(err))
		return models.Cart{}, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var itemsByCartId []models.CartItem
	for rows.Next() {
		var tmpItem models.CartItem
		if err := rows.Scan(&tmpItem.Id, &tmpItem.CartId, &tmpItem.Product, &tmpItem.Quantity); err != nil {
			log.Error("Failed to scan row", sl.Err(err))
			continue
		}
		itemsByCartId = append(itemsByCartId, tmpItem)
	}

	return models.Cart{
		Id:    cartId,
		Items: itemsByCartId,
	}, nil
}
