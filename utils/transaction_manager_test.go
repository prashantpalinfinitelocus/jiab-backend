package utils

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// fakeDriver is a minimal database/sql driver that only counts transaction
// outcomes, so the pool and rollback behaviour can be tested without a database.
type fakeDriver struct{ begins, commits, rollbacks atomic.Int32 }

func (d *fakeDriver) Open(string) (driver.Conn, error) { return &fakeConn{d}, nil }

type fakeConn struct{ d *fakeDriver }

func (c *fakeConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not supported") }
func (c *fakeConn) Close() error                        { return nil }
func (c *fakeConn) Begin() (driver.Tx, error)           { c.d.begins.Add(1); return &fakeTx{c.d}, nil }

type fakeTx struct{ d *fakeDriver }

func (t *fakeTx) Commit() error   { t.d.commits.Add(1); return nil }
func (t *fakeTx) Rollback() error { t.d.rollbacks.Add(1); return nil }

var fakeDriverSeq atomic.Int32

// newTestManager returns a manager over a pool capped at ONE connection: if a
// transaction is ever leaked, the next StartTxn blocks - the same way the real
// 100-connection pool ran dry on staging.
func newTestManager(t *testing.T) (*TransactionManager, *fakeDriver) {
	t.Helper()
	d := &fakeDriver{}
	name := "fake-txn-" + string(rune('a'+fakeDriverSeq.Add(1)))
	sql.Register(name, d)
	sqlDB, err := sql.Open(name, "")
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{DisableAutomaticPing: true, SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}
	return NewTransactionManager(gdb), d
}

func mustStartWithin(t *testing.T, m *TransactionManager, d time.Duration) *gorm.DB {
	t.Helper()
	type res struct {
		tx  *gorm.DB
		err error
	}
	ch := make(chan res, 1)
	go func() { tx, err := m.StartTxn(); ch <- res{tx, err} }()
	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatal(r.err)
		}
		return r.tx
	case <-time.After(d):
		t.Fatal("StartTxn blocked: a previous transaction leaked its connection")
		return nil
	}
}

// The regression: a function that opens a transaction and returns early WITHOUT
// calling AbortTxn must still give its connection back.
func TestEarlyReturnWithoutAbortDoesNotLeakConnection(t *testing.T) {
	m, d := newTestManager(t)
	leaky := func() error {
		tx, err := m.StartTxn()
		if err != nil {
			return err
		}
		defer m.RollbackOnPanic(tx)
		return errors.New("early return, no AbortTxn") // the old leak
	}
	for i := 0; i < 5; i++ {
		_ = leaky()
	}
	tx := mustStartWithin(t, m, 2*time.Second)
	m.AbortTxn(tx)
	if got := d.rollbacks.Load(); got != 6 {
		t.Fatalf("rollbacks = %d, want 6", got)
	}
}

func TestPanicRollsBackAndRepanics(t *testing.T) {
	m, d := newTestManager(t)
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("panic was swallowed")
			}
		}()
		tx, _ := m.StartTxn()
		defer m.RollbackOnPanic(tx)
		panic("boom")
	}()
	if d.rollbacks.Load() != 1 {
		t.Fatalf("rollbacks = %d, want 1", d.rollbacks.Load())
	}
	m.AbortTxn(mustStartWithin(t, m, 2*time.Second))
}

func TestCommittedTransactionIsNotRolledBack(t *testing.T) {
	m, d := newTestManager(t)
	func() {
		tx, _ := m.StartTxn()
		defer m.RollbackOnPanic(tx)
		m.CommitTxn(tx)
	}()
	if d.commits.Load() != 1 || d.rollbacks.Load() != 0 {
		t.Fatalf("commits=%d rollbacks=%d, want 1 and 0", d.commits.Load(), d.rollbacks.Load())
	}
}

// Backstop: even a path that forgets everything gets its connection back when
// the transaction's lifetime context expires.
func TestLifetimeBackstopReleasesForgottenTransaction(t *testing.T) {
	old := txnMaxLifetime
	txnMaxLifetime = 200 * time.Millisecond
	defer func() { txnMaxLifetime = old }()

	m, d := newTestManager(t)
	_, _ = m.StartTxn()  // forgotten: never committed, aborted or deferred
	txnMaxLifetime = old // the waiter below must not inherit the short lifetime
	tx := mustStartWithin(t, m, 3*time.Second)
	m.AbortTxn(tx)
	if d.rollbacks.Load() < 1 {
		t.Fatal("expiry did not roll the forgotten transaction back")
	}
}
