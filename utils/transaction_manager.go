package utils

import (
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"gorm.io/gorm"
)

type TransactionManager struct {
	db *gorm.DB
}

func NewTransactionManager(db *gorm.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

func (t *TransactionManager) StartTxn() (*gorm.DB, error) {
	tx := t.db.Begin()
	if tx.Error != nil {
		return nil, errs.NewInternalServerError("error starting transaction", tx.Error)
	}
	return tx, nil
}

func (t *TransactionManager) CommitTxn(tx *gorm.DB) {
	if tx != nil {
		_ = tx.Commit()
	}
}

func (t *TransactionManager) AbortTxn(tx *gorm.DB) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

func (t *TransactionManager) RollbackOnPanic(tx *gorm.DB) {
	if r := recover(); r != nil {
		t.AbortTxn(tx)
		panic(r)
	}
}
