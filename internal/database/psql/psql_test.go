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

func TestContextCanceled(t *testing.T) {
	t.Run("CreateCart", func(t *testing.T) {
		storage, mock, cleanup := newTestStorage(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := storage.CreateCart(ctx)
		assert.ErrorIs(t, err, context.Canceled)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("AddToCart", func(t *testing.T) {
		storage, mock, cleanup := newTestStorage(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		cartId := 1
		item := models.CartItem{
			Product:  "product",
			Quantity: 2,
		}

		_, err := storage.AddToCart(ctx, cartId, item)
		assert.ErrorIs(t, err, context.Canceled)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("RemoveFromCart", func(t *testing.T) {
		storage, mock, cleanup := newTestStorage(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := storage.RemoveFromCart(ctx, 1, 1)
		assert.ErrorIs(t, err, context.Canceled)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ViewCart", func(t *testing.T) {
		storage, mock, cleanup := newTestStorage(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := storage.ViewCart(ctx, 1)
		assert.ErrorIs(t, err, context.Canceled)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDeadlineExceeded(t *testing.T) {
	t.Run("CreateCart", func(t *testing.T) {
		storage, mock, cleanup := newTestStorage(t)
		defer cleanup()

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()

		time.Sleep(time.Millisecond * 15)

		_, err := storage.CreateCart(ctx)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("AddToCart", func(t *testing.T) {
		storage, mock, cleanup := newTestStorage(t)
		defer cleanup()

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()

		time.Sleep(time.Millisecond * 15)

		cartId := 1
		item := models.CartItem{
			Product:  "product",
			Quantity: 2,
		}

		_, err := storage.AddToCart(ctx, cartId, item)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("RemoveFromCart", func(t *testing.T) {
		storage, mock, cleanup := newTestStorage(t)
		defer cleanup()

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()

		time.Sleep(time.Millisecond * 15)

		err := storage.RemoveFromCart(ctx, 1, 1)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ViewCart", func(t *testing.T) {
		storage, mock, cleanup := newTestStorage(t)
		defer cleanup()

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()

		time.Sleep(time.Millisecond * 15)

		_, err := storage.ViewCart(ctx, 1)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCreateCart(t *testing.T) {
	tests := []struct {
		name       string
		setupMock  func(sqlmock.Sqlmock)
		expectCart models.Cart
		expectErr  error
	}{
		{
			name: "Success",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id"}).AddRow(123)
				mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO cart DEFAULT VALUES RETURNING id")).WillReturnRows(rows)
			},
			expectCart: models.Cart{Id: 123},
			expectErr:  nil,
		},
		{
			name: "Query Error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO cart DEFAULT VALUES RETURNING id")).WillReturnError(errors.New("db error"))
			},
			expectCart: models.Cart{},
			expectErr:  errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, mock, cleanup := newTestStorage(t)
			defer cleanup()

			tt.setupMock(mock)

			cart, err := storage.CreateCart(context.Background())
			if tt.expectErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectCart, cart)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAddToCart(t *testing.T) {
	tests := []struct {
		name      string
		cartId    int
		item      models.CartItem
		setupMock func(mock sqlmock.Sqlmock)
		wantErr   error
		wantItem  models.CartItem
	}{
		{
			name:   "Success",
			cartId: 1,
			item: models.CartItem{
				Product:  "product",
				Quantity: 2,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()

				rowsCart := sqlmock.NewRows([]string{"id"}).AddRow(1)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM cart WHERE id=$1`)).
					WithArgs(1).
					WillReturnRows(rowsCart)

				rowsItem := sqlmock.NewRows([]string{"id"}).AddRow(10)
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO item (cart_id, product, quantity) VALUES ($1, $2, $3) RETURNING id;`)).
					WithArgs(1, "product", 2).
					WillReturnRows(rowsItem)

				mock.ExpectCommit()
			},
			wantErr:  nil,
			wantItem: models.CartItem{Id: 10, CartId: 1, Product: "product", Quantity: 2},
		},
		{
			name:   "CartNotFound",
			cartId: 1,
			item: models.CartItem{
				Product:  "product",
				Quantity: 2,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM cart WHERE id=$1`)).
					WithArgs(1).
					WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
			wantErr: databaseerrors.ErrNotFound,
		},
		{
			name:   "InsertItemFail",
			cartId: 1,
			item: models.CartItem{
				Product:  "product",
				Quantity: 2,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				rowsCart := sqlmock.NewRows([]string{"id"}).AddRow(1)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM cart WHERE id=$1`)).
					WithArgs(1).
					WillReturnRows(rowsCart)
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO item (cart_id, product, quantity) VALUES ($1, $2, $3) RETURNING id;`)).
					WithArgs(1, "product", 2).
					WillReturnError(errors.New("insert item error"))
				mock.ExpectRollback()
			},
			wantErr: errors.New("insert item error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, mock, cleanup := newTestStorage(t)
			defer cleanup()

			tt.setupMock(mock)
			gotItem, err := storage.AddToCart(context.Background(), tt.cartId, tt.item)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantItem, gotItem)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRemoveFromCart(t *testing.T) {
	tests := []struct {
		name      string
		cartId    int
		itemId    int
		setupMock func(mock sqlmock.Sqlmock)
		wantErr   error
	}{
		{
			name:   "Success",
			cartId: 10,
			itemId: 20,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()

				rowsCart := sqlmock.NewRows([]string{"id"}).AddRow(10)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM cart WHERE id=$1;`)).
					WithArgs(10).
					WillReturnRows(rowsCart)

				rowsItem := sqlmock.NewRows([]string{"cart_id"}).AddRow(10)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT cart_id FROM item WHERE id=$1;`)).
					WithArgs(20).
					WillReturnRows(rowsItem)

				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM item WHERE id=$1;`)).
					WithArgs(20).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectCommit()
			},
			wantErr: nil,
		},
		{
			name:   "ItemNotFound",
			cartId: 10,
			itemId: 20,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()

				rowsCart := sqlmock.NewRows([]string{"id"}).AddRow(10)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM cart WHERE id=$1;`)).
					WithArgs(10).
					WillReturnRows(rowsCart)

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT cart_id FROM item WHERE id=$1;`)).
					WithArgs(20).
					WillReturnError(sql.ErrNoRows)

				mock.ExpectRollback()
			},
			wantErr: databaseerrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, mock, cleanup := newTestStorage(t)
			defer cleanup()

			tt.setupMock(mock)
			err := storage.RemoveFromCart(context.Background(), tt.cartId, tt.itemId)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestViewCart(t *testing.T) {
	tests := []struct {
		name      string
		cartId    int
		setupMock func(mock sqlmock.Sqlmock)
		wantErr   error
		wantCart  models.Cart
	}{
		{
			name:   "Success",
			cartId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`
                    SELECT COUNT(*) FROM cart WHERE id=$1;
                `)).WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

				rows := sqlmock.NewRows([]string{"id", "cart_id", "product", "quantity"}).
					AddRow(11, 1, "apple", 3).
					AddRow(12, 1, "banana", 5)

				mock.ExpectQuery(regexp.QuoteMeta(`
                    SELECT id, cart_id, product, quantity FROM item WHERE cart_id=$1;
                `)).WithArgs(1).WillReturnRows(rows)
			},
			wantErr: nil,
			wantCart: models.Cart{
				Id: 1,
				Items: []models.CartItem{
					{Id: 11, CartId: 1, Product: "apple", Quantity: 3},
					{Id: 12, CartId: 1, Product: "banana", Quantity: 5},
				},
			},
		},
		{
			name:   "CartNotFound",
			cartId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`
                    SELECT COUNT(*) FROM cart WHERE id=$1;
                `)).WithArgs(1).WillReturnError(databaseerrors.ErrNotFound)
			},
			wantErr: databaseerrors.ErrNotFound,
		},
		{
			name:   "QueryError",
			cartId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`
                    SELECT COUNT(*) FROM cart WHERE id=$1;
                `)).WithArgs(1).WillReturnError(errors.New("query error"))
			},
			wantErr: errors.New("query error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, mock, cleanup := newTestStorage(t)
			defer cleanup()

			tt.setupMock(mock)

			cart, err := storage.ViewCart(context.Background(), tt.cartId)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCart, cart)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
