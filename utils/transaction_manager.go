package utils

import (
	"context"
	"sync"
	"time"

	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"gorm.io/gorm"
)

// txnMaxLifetime is a backstop, not a working limit: it matches Cloud Run's
// 300s request timeout, so no request can legitimately keep a transaction
// open longer. If a code path ever forgets to commit or roll back, database/sql
// rolls the transaction back when this context expires and the pooled
// connection is released, instead of being lost for the life of the process.
var txnMaxLifetime = 5 * time.Minute

type TransactionManager struct {
	db *gorm.DB
	// cancels holds the cancel func of each open transaction's lifetime
	// context, keyed by the *gorm.DB StartTxn returned.
	cancels sync.Map
}

func NewTransactionManager(db *gorm.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

func (t *TransactionManager) StartTxn() (*gorm.DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), txnMaxLifetime)
	tx := t.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		cancel()
		return nil, errs.NewInternalServerError("error starting transaction", tx.Error)
	}
	t.cancels.Store(tx, cancel)
	return tx, nil
}

// release frees the lifetime context of a finished transaction.
func (t *TransactionManager) release(tx *gorm.DB) {
	if cancel, ok := t.cancels.LoadAndDelete(tx); ok {
		cancel.(context.CancelFunc)()
	}
}

func (t *TransactionManager) CommitTxn(tx *gorm.DB) {
	if tx != nil {
		_ = tx.Commit()
		t.release(tx)
	}
}

func (t *TransactionManager) AbortTxn(tx *gorm.DB) {
	if tx != nil {
		_ = tx.Rollback()
		t.release(tx)
	}
}

// RollbackOnPanic must be deferred immediately after StartTxn. On every exit
// path it rolls the transaction back; that is a harmless no-op when the
// transaction was already committed or aborted (database/sql returns
// ErrTxDone, which is ignored). This is what guarantees the pooled connection
// goes back to the pool on early returns and panics alike - without it, every
// error return that skipped AbortTxn leaked a connection permanently, and
// ~100 such requests exhausted the pool and hung every database call.
func (t *TransactionManager) RollbackOnPanic(tx *gorm.DB) {
	r := recover()
	t.AbortTxn(tx)
	if r != nil {
		panic(r)
	}
}
