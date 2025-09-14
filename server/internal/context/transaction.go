package context

import (
	"context"

	"gorm.io/gorm"
)

type contextKey string

const (
	TRANSACTION_KEY contextKey = "transaction"
)

// GetTransaction retrieves a transaction from the context
func GetTransaction(ctx context.Context) (*gorm.DB, bool) {
	tx, ok := ctx.Value(TRANSACTION_KEY).(*gorm.DB)
	return tx, ok
}

// WithTransaction adds a transaction to the context
func WithTransaction(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, TRANSACTION_KEY, tx)
}