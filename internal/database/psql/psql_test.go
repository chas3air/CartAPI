package psql_test

import (
	databaseerrors "cartapi/internal/database"
	"cartapi/internal/database/psql"
	"cartapi/internal/models"
	"cartapi/pkg/lib/logger/slogdiscard"

	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
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
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestCreateCart_DeadlineExceeded(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
	defer func() {
		cancel()
	}()

	time.Sleep(time.Millisecond * 55)
	_, err := storage.CreateCart(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
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
func TestAddToCart_DeadlineExceeded(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
	defer func() {
		cancel()
	}()

	cart_id := 1
	item := models.CartItem{
		Product:  "product",
		Quantity: 2,
	}

	time.Sleep(time.Millisecond * 55)
	_, err := storage.AddToCart(ctx, cart_id, item)
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
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

func TestRemoveFromCart_Success(t *testing.T) {
	s, mock, close := newTestStorage(t)
	defer close()

	ctx := context.Background()
	cartId := 10
	itemId := 20

	mock.ExpectBegin()

	// Проверка существования корзины
	rows := sqlmock.NewRows([]string{"id"}).AddRow(cartId)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM cart WHERE id=$1;`)).
		WithArgs(cartId).
		WillReturnRows(rows)

	// Проверка сузествования айтема
	rows = sqlmock.NewRows([]string{"id"}).AddRow(itemId)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM item WHERE id=$1;`)).
		WithArgs(itemId).
		WillReturnRows(rows)

	// Удаление из связующей таблицы
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM cart_item WHERE cart_id=$1 AND item_id=$2;`)).
		WithArgs(cartId, itemId).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Удаление из таблицы items
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM item WHERE id=$1;`)).
		WithArgs(itemId).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	err := s.RemoveFromCart(ctx, cartId, itemId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRemoveFromCart_ItemNotFound(t *testing.T) {
	s, mock, close := newTestStorage(t)
	defer close()

	ctx := context.Background()
	cartId := 10
	itemId := 20

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM cart WHERE id=$1;`)).
		WithArgs(cartId).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(cartId))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM item WHERE id=$1;`)).
		WithArgs(itemId).
		WillReturnError(sql.ErrNoRows)

	mock.ExpectRollback()

	err := s.RemoveFromCart(ctx, cartId, itemId)
	if !errors.Is(err, databaseerrors.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestRemoveFromCart_ContextCanceled(t *testing.T) {
	s, _, close := newTestStorage(t)
	defer close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // сразу отменяем контекст

	err := s.RemoveFromCart(ctx, 1, 1)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled error, got %v", err)
	}
}

func TestRemoveFromCart_TransactionBeginFail(t *testing.T) {
	s, mock, close := newTestStorage(t)
	defer close()

	ctx := context.Background()
	cartId := 10
	itemId := 20

	mock.ExpectBegin().WillReturnError(errors.New("begin error"))

	err := s.RemoveFromCart(ctx, cartId, itemId)
	if err == nil || err.Error() != "database.psql.RemoveFromCart: begin error" {
		t.Fatalf("expected begin error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestRemoveFromCart_DeadlineExceeced(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
	defer func() {
		cancel()
	}()

	time.Sleep(time.Millisecond * 55)
	err := storage.RemoveFromCart(ctx, 0, 0)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestRemoveFromCart_CartNotFound(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	cartId := 1
	itemId := 1

	mock.ExpectBegin()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM cart WHERE id=$1`)).
		WithArgs(cartId).
		WillReturnError(sql.ErrNoRows)

	mock.ExpectRollback()

	err := storage.RemoveFromCart(ctx, cartId, itemId)
	assert.ErrorIs(t, err, databaseerrors.ErrNotFound)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestViewCart_ContextCanceled(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := storage.ViewCart(ctx, 0)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestViewCart_DeadlineExceeded(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
	defer func() {
		cancel()
	}()

	time.Sleep(time.Millisecond * 55)
	_, err := storage.ViewCart(ctx, 0)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestViewCart_QueryError(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT ci.item_id, ci.cart_id, i.product, i.quantity FROM cart_item AS ci
        JOIN item AS i
        ON ci.item_id = i.id
        WHERE ci.cart_id=$1;
    `)).WithArgs(1).WillReturnError(errors.New("query failure"))

	_, err := storage.ViewCart(ctx, 1)
	if err == nil {
		t.Fatal("expected error on query failure, got nil")
	}
	if err.Error() != "database.psql.ViewCart: query failure" {
		t.Errorf("unexpected error message: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestViewCart_ScanRowError(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()
	ctx := context.Background()

	// Добавление строк в таблицу айтемов
	rows := sqlmock.NewRows([]string{"item_id", "cart_id", "product", "quantity"}).
		AddRow("not_an_int", 1, "apple", 3)
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT ci.item_id, ci.cart_id, i.product, i.quantity FROM cart_item AS ci
        JOIN item AS i
        ON ci.item_id = i.id
        WHERE ci.cart_id=$1;
    `)).WithArgs(1).WillReturnRows(rows)

	cart, err := storage.ViewCart(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cart.Items) != 0 {
		t.Errorf("expected 0 items due to scan error, got %d", len(cart.Items))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestViewCart_Success(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"item_id", "cart_id", "product", "quantity"}).
		AddRow(11, 1, "apple", 3).
		AddRow(12, 1, "banana", 5)

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT ci.item_id, ci.cart_id, i.product, i.quantity FROM cart_item AS ci
        JOIN item AS i
        ON ci.item_id = i.id
        WHERE ci.cart_id=$1;
    `)).WithArgs(1).WillReturnRows(rows)

	cart, err := storage.ViewCart(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cart.Id != 1 {
		t.Errorf("expected cart id 1, got %d", cart.Id)
	}
	if len(cart.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(cart.Items))
	}
	if cart.Items[0].Product != "apple" || cart.Items[1].Product != "banana" {
		t.Errorf("unexpected products in cart items")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}
