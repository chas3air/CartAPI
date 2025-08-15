package psql_test

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	databaseerrors "cartapi/internal/database"
	"cartapi/internal/database/psql"
	"cartapi/internal/models"
	"cartapi/pkg/lib/logger/slogdiscard"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func newTestStorage(t *testing.T) (*psql.Storage, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %s", err)
	}
	storage := psql.NewWithParams(slogdiscard.NewDiscardLogger(), &sqlx.DB{DB: db})
	cleanup := func() { db.Close() }
	return storage, mock, cleanup
}

func TestCreateCart(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()

	tests := []struct {
		name       string
		setupMock  func(sqlmock.Sqlmock)
		ctx        context.Context
		expectCart models.Cart
		expectErr  error
	}{
		{
			name: "Success",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id"}).AddRow(123)
				mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO cart DEFAULT VALUES RETURNING id")).WillReturnRows(rows)
			},
			ctx:        context.Background(),
			expectCart: models.Cart{Id: 123},
			expectErr:  nil,
		},
		{
			name:      "Context canceled",
			setupMock: func(sqlmock.Sqlmock) {},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			expectErr: context.Canceled,
		},
		{
			name:      "Deadline exceeded",
			setupMock: func(sqlmock.Sqlmock) {},
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
				time.Sleep(15 * time.Millisecond)
				cancel()
				return ctx
			}(),
			expectErr: context.DeadlineExceeded,
		},
		{
			name: "Query error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO cart DEFAULT VALUES RETURNING id")).WillReturnError(errors.New("db error"))
			},
			ctx:        context.Background(),
			expectCart: models.Cart{},
			expectErr:  errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)
			cart, err := storage.CreateCart(tt.ctx)
			if tt.expectErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectCart, cart)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAddToCart(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()

	tests := []struct {
		name      string
		cartId    int
		item      models.CartItem
		setupMock func(sqlmock.Sqlmock)
		ctx       context.Context
		wantItem  models.CartItem
		wantErr   error
	}{
		{
			name:   "Success",
			cartId: 1,
			item:   models.CartItem{Product: "product", Quantity: 2},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM cart WHERE id=$1`)).
					WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO item (cart_id, product, quantity) VALUES ($1, $2, $3) RETURNING id;`)).
					WithArgs(1, "product", 2).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))
				mock.ExpectCommit()
			},
			ctx:      context.Background(),
			wantItem: models.CartItem{Id: 10, CartId: 1, Product: "product", Quantity: 2},
			wantErr:  nil,
		},
		{
			name:      "Context canceled",
			cartId:    1,
			item:      models.CartItem{},
			setupMock: func(sqlmock.Sqlmock) {},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			wantErr: context.Canceled,
		},
		{
			name:      "Deadline exceeded",
			cartId:    1,
			item:      models.CartItem{},
			setupMock: func(sqlmock.Sqlmock) {},
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
				time.Sleep(15 * time.Millisecond)
				cancel()
				return ctx
			}(),
			wantErr: context.DeadlineExceeded,
		},
		{
			name:   "Cart not found",
			cartId: 1,
			item:   models.CartItem{Product: "product", Quantity: 2},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM cart WHERE id=$1`)).
					WithArgs(1).WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
			ctx:     context.Background(),
			wantErr: databaseerrors.ErrNotFound,
		},
		{
			name:   "Insert item error",
			cartId: 1,
			item:   models.CartItem{Product: "product", Quantity: 2},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM cart WHERE id=$1`)).
					WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO item (cart_id, product, quantity) VALUES ($1, $2, $3) RETURNING id;`)).
					WithArgs(1, "product", 2).WillReturnError(errors.New("insert item error"))
				mock.ExpectRollback()
			},
			ctx:     context.Background(),
			wantErr: errors.New("insert item error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)
			gotItem, err := storage.AddToCart(tt.ctx, tt.cartId, tt.item)

			if tt.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantItem, gotItem)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRemoveFromCart(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()

	tests := []struct {
		name      string
		cartId    int
		itemId    int
		setupMock func(sqlmock.Sqlmock)
		ctx       context.Context
		wantErr   error
	}{
		{
			name:   "Success",
			cartId: 10,
			itemId: 20,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM cart WHERE id=$1;`)).WithArgs(10).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT cart_id FROM item WHERE id=$1;`)).WithArgs(20).
					WillReturnRows(sqlmock.NewRows([]string{"cart_id"}).AddRow(10))
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM item WHERE id=$1;`)).WithArgs(20).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			ctx:     context.Background(),
			wantErr: nil,
		},
		{
			name:      "Context canceled",
			cartId:    1,
			itemId:    1,
			setupMock: func(sqlmock.Sqlmock) {},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			wantErr: context.Canceled,
		},
		{
			name:      "Deadline exceeded",
			cartId:    1,
			itemId:    1,
			setupMock: func(sqlmock.Sqlmock) {},
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
				time.Sleep(15 * time.Millisecond)
				cancel()
				return ctx
			}(),
			wantErr: context.DeadlineExceeded,
		},
		{
			name:   "Item not found",
			cartId: 10,
			itemId: 20,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM cart WHERE id=$1;`)).WithArgs(10).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT cart_id FROM item WHERE id=$1;`)).
					WithArgs(20).WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
			ctx:     context.Background(),
			wantErr: databaseerrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)
			err := storage.RemoveFromCart(tt.ctx, tt.cartId, tt.itemId)

			if tt.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestViewCart(t *testing.T) {
	storage, mock, cleanup := newTestStorage(t)
	defer cleanup()

	tests := []struct {
		name      string
		cartId    int
		setupMock func(sqlmock.Sqlmock)
		ctx       context.Context
		wantCart  models.Cart
		wantErr   error
	}{
		{
			name:   "Success",
			cartId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM cart WHERE id=$1;`)).WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				rows := sqlmock.NewRows([]string{"id", "cart_id", "product", "quantity"}).
					AddRow(11, 1, "apple", 3).
					AddRow(12, 1, "banana", 5)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, cart_id, product, quantity FROM item WHERE cart_id=$1;`)).
					WithArgs(1).WillReturnRows(rows)
			},
			ctx: context.Background(),
			wantCart: models.Cart{
				Id: 1,
				Items: []models.CartItem{
					{Id: 11, CartId: 1, Product: "apple", Quantity: 3},
					{Id: 12, CartId: 1, Product: "banana", Quantity: 5},
				},
			},
			wantErr: nil,
		},
		{
			name:      "Context canceled",
			cartId:    1,
			setupMock: func(sqlmock.Sqlmock) {},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			wantErr: context.Canceled,
		},
		{
			name:      "Deadline exceeded",
			cartId:    1,
			setupMock: func(sqlmock.Sqlmock) {},
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
				time.Sleep(15 * time.Millisecond)
				cancel()
				return ctx
			}(),
			wantErr: context.DeadlineExceeded,
		},
		{
			name:   "Cart not found",
			cartId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM cart WHERE id=$1;`)).
					WithArgs(1).WillReturnError(databaseerrors.ErrNotFound)
			},
			ctx:     context.Background(),
			wantErr: databaseerrors.ErrNotFound,
		},
		{
			name:   "Query error",
			cartId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM cart WHERE id=$1;`)).
					WithArgs(1).WillReturnError(errors.New("query error"))
			},
			ctx:     context.Background(),
			wantErr: errors.New("query error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)
			cart, err := storage.ViewCart(tt.ctx, tt.cartId)

			if tt.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCart, cart)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
