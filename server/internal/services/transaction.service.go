package services

import (
	"context"
	"fmt"
	"waugzee/internal/database"
	logger "github.com/Bparsons0904/goLogger"

	"gorm.io/gorm"
)

// TransactionService handles database transactions using context injection
type TransactionService struct {
	db  database.DB
	log logger.Logger
}

func NewTransactionService(db database.DB) *TransactionService {
	return &TransactionService{
		db:  db,
		log: logger.New("TransactionService"),
	}
}

// Execute runs the provided function within a database transaction
// The function receives a context and the transaction DB instance
// Automatically handles commit/rollback based on function result
// Panics are converted to errors unless rollback fails (which crashes service for data safety)
func (ts *TransactionService) Execute(
	ctx context.Context,
	fn func(context.Context, *gorm.DB) error,
) (err error) {
	log := ts.log.Function("Execute")
	log.Info("Executing transaction")

	// Begin transaction
	tx := ts.db.SQLWithContext(ctx).Begin()
	if tx.Error != nil {
		return log.Err("failed to begin transaction", tx.Error)
	}

	// Handle panics with intelligent rollback
	defer func() {
		if r := recover(); r != nil {
			panicErr := log.ErrMsg("panic during transaction: " + fmt.Sprintf("%v", r))
			log.Er("panic during transaction, rolling back", panicErr)

			if rollbackErr := tx.Rollback().Error; rollbackErr != nil {
				// Critical failure - data integrity at risk, crash service
				log.Er("CRITICAL: failed to rollback after panic", rollbackErr, "panic", r)
				panic(
					fmt.Sprintf(
						"transaction rollback failed: %v (original panic: %v)",
						rollbackErr,
						r,
					),
				)
			}

			// Rollback succeeded - convert panic to error for graceful handling
			log.Info("transaction rolled back successfully after panic")
			err = panicErr
		}
	}()

	// Execute the function with the transaction
	if err = fn(ctx, tx); err != nil {
		if rollbackErr := tx.Rollback().Error; rollbackErr != nil {
			log.Er("CRITICAL: failed to rollback after function error", rollbackErr, "originalError", err)
			return log.Error("transaction rollback failed", "rollbackError", rollbackErr, "originalError", err)
		}
		return err
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return log.Err("failed to commit transaction", err)
	}

	// Removed success logging to reduce log noise during bulk operations
	return nil
}

// ExecuteInTransaction is an alias for Execute for backward compatibility
func (ts *TransactionService) ExecuteInTransaction(
	ctx context.Context,
	fn func(context.Context, *gorm.DB) error,
) error {
	return ts.Execute(ctx, fn)
}
