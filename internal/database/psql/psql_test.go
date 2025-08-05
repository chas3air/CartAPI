package psql_test

import (
	databaseerrors "cartapi/internal/database"
	"cartapi/internal/database/psql"
	"cartapi/internal/models"
	"cartapi/pkg/lib/logger/handler/slogdiscard"
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func newTestStorage(t *testing.T) (*psql.Storage, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %s", err)
	}

	storage := psql.NewWithParams(slogdiscard.NewDiscardLogger(), &sqlx.DB{
		DB: db,
	})
	cleanup := func() { db.Close() }
	return storage, mock, cleanup
}

func TestCreateCart_ContextCanceled(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := storage.CreateCart(ctx)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestCreateCart_Success(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()

	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id"}).AddRow(123)
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO cart DEFAULT VALUES RETURNING id")).
		WillReturnRows(rows)

	cart, err := storage.CreateCart(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 123, cart.Id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateCart_QueryError(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()

	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO cart DEFAULT VALUES RETURNING id")).
		WillReturnError(errors.New("db error"))

	cart, err := storage.CreateCart(ctx)
	assert.Error(t, err)
	assert.Equal(t, models.Cart{}, cart)
	assert.NoError(t, mock.ExpectationsWereMet())

}

func TestAddToCart_ContextCanceled(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cart_id := 1
	item := models.CartItem{
		Product:  "product",
		Quantity: 2,
	}

	_, err := storage.AddToCart(ctx, cart_id, item)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestAddToCart_Success(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	cartId := 1
	item := models.CartItem{
		Id:       10,
		Product:  "product",
		Quantity: 2,
	}

	mock.ExpectBegin()

	// Создание корзины
	rowsCart := sqlmock.NewRows([]string{"id"}).AddRow(cartId)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM cart WHERE id=$1`)).
		WithArgs(cartId).
		WillReturnRows(rowsCart)

	// Вставка айтема
	rowsItem := sqlmock.NewRows([]string{"id"}).AddRow(item.Id)
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO item (product, quantity) VALUES ($1, $2) RETURNING id;`)).
		WithArgs(item.Product, item.Quantity).
		WillReturnRows(rowsItem)

	// Вставка связи
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO cart_item (cart_id, item_id) VALUES ($1, $2);`)).
		WithArgs(cartId, item.Id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	result, err := storage.AddToCart(ctx, cartId, item)
	assert.NoError(t, err)
	assert.Equal(t, item.Id, result.Id)
	assert.Equal(t, cartId, result.CartId)
	assert.Equal(t, item.Product, result.Product)
	assert.Equal(t, item.Quantity, result.Quantity)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddToCart_CartNotFound(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	cartId := 1
	item := models.CartItem{
		Id:       10,
		Product:  "product",
		Quantity: 2,
	}

	mock.ExpectBegin()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM cart WHERE id=$1`)).
		WithArgs(cartId).
		WillReturnError(sql.ErrNoRows)

	mock.ExpectRollback()

	_, err := storage.AddToCart(ctx, cartId, item)
	assert.ErrorIs(t, err, databaseerrors.ErrNotFound)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddToCart_InsertItemFail(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	cartId := 1
	item := models.CartItem{
		Id:       10,
		Product:  "product",
		Quantity: 2,
	}

	mock.ExpectBegin()

	rowsCart := sqlmock.NewRows([]string{"id"}).AddRow(cartId)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM cart WHERE id=$1`)).
		WithArgs(cartId).
		WillReturnRows(rowsCart)

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO item (product, quantity) VALUES ($1, $2) RETURNING id;`)).
		WithArgs(item.Product, item.Quantity).
		WillReturnError(errors.New("insert item error"))

	mock.ExpectRollback()

	_, err := storage.AddToCart(ctx, cartId, item)
	assert.Error(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddToCart_InsertRelationFail(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	cartId := 1
	item := models.CartItem{
		Id:       10,
		Product:  "product",
		Quantity: 2,
	}

	mock.ExpectBegin()

	rowsCart := sqlmock.NewRows([]string{"id"}).AddRow(cartId)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM cart WHERE id=$1`)).
		WithArgs(cartId).
		WillReturnRows(rowsCart)

	rowsItem := sqlmock.NewRows([]string{"id"}).AddRow(item.Id)
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO item (product, quantity) VALUES ($1, $2) RETURNING id;`)).
		WithArgs(item.Product, item.Quantity).
		WillReturnRows(rowsItem)

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO cart_item (cart_id, item_id) VALUES ($1, $2);`)).
		WithArgs(cartId, item.Id).
		WillReturnError(errors.New("insert relation error"))

	mock.ExpectRollback()

	_, err := storage.AddToCart(ctx, cartId, item)
	assert.Error(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}
