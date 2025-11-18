package services

import (
	"context"
	"errors"
	"testing"
	"waugzee/internal/database"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	return gormDB, mock
}

func TestTransactionService_Execute_Success(t *testing.T) {
	gormDB, mock := setupTestDB(t)

	mock.ExpectBegin()
	mock.ExpectCommit()

	dbWrapper := database.DB{SQL: gormDB}
	service := NewTransactionService(dbWrapper)

	called := false
	err := service.Execute(context.Background(), func(ctx context.Context, tx *gorm.DB) error {
		called = true
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, called)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionService_Execute_RollbackOnError(t *testing.T) {
	gormDB, mock := setupTestDB(t)

	mock.ExpectBegin()
	mock.ExpectRollback()

	dbWrapper := database.DB{SQL: gormDB}
	service := NewTransactionService(dbWrapper)

	expectedError := errors.New("test error")
	err := service.Execute(context.Background(), func(ctx context.Context, tx *gorm.DB) error {
		return expectedError
	})

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionService_Execute_PanicRecovery(t *testing.T) {
	gormDB, mock := setupTestDB(t)

	mock.ExpectBegin()
	mock.ExpectRollback()

	dbWrapper := database.DB{SQL: gormDB}
	service := NewTransactionService(dbWrapper)

	err := service.Execute(context.Background(), func(ctx context.Context, tx *gorm.DB) error {
		panic("test panic")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "panic during transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}
