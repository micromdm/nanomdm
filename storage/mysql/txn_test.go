package mysql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/micromdm/nanomdm/storage/mysql/sqlc"
)

// newTxnMock returns a database handle backed by [txnMock], for setting
// transaction expectations.
func newTxnMock(t *testing.T) (*sql.DB, *txnMock) {
	t.Helper()
	m := new(txnMock)
	db := sql.OpenDB(m)
	t.Cleanup(func() { db.Close() })
	return db, m
}

func noop(_ context.Context, _ *sql.Tx, _ *sqlc.Queries) error { return nil }

func TestExecCommitsOnSuccess(t *testing.T) {
	db, mock := newTxnMock(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	if err := NewTxn(db, sqlc.New(db)).Exec(context.Background(), noop); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestExecRollsBackOnFnError(t *testing.T) {
	db, mock := newTxnMock(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	fnErr := errors.New("bad query")
	err := NewTxn(db, sqlc.New(db)).Exec(context.Background(), func(_ context.Context, _ *sql.Tx, _ *sqlc.Queries) error {
		return fnErr
	})

	if !errors.Is(err, fnErr) {
		t.Fatalf("got %v, want %v", err, fnErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestExecReturnsCommitError(t *testing.T) {
	db, mock := newTxnMock(t)
	mock.ExpectBegin()
	commitErr := errors.New("commit failed")
	mock.ExpectCommit().WillReturnError(commitErr)

	err := NewTxn(db, sqlc.New(db)).Exec(context.Background(), noop)

	if !errors.Is(err, commitErr) {
		t.Fatalf("got %v, want %v", err, commitErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestExecBeginError(t *testing.T) {
	db, mock := newTxnMock(t)
	beginErr := errors.New("begin failed")
	mock.ExpectBegin().WillReturnError(beginErr)

	err := NewTxn(db, sqlc.New(db)).Exec(context.Background(), noop)

	if !errors.Is(err, beginErr) {
		t.Fatalf("got %v, want %v", err, beginErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestExecRecoversPanic(t *testing.T) {
	db, mock := newTxnMock(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		if r != "boom" {
			t.Fatalf("got panic %v, want \"boom\"", r)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	}()

	NewTxn(db, sqlc.New(db)).Exec(context.Background(), func(_ context.Context, _ *sql.Tx, _ *sqlc.Queries) error {
		panic("boom")
	})
}

func TestExecPanicRollbackError(t *testing.T) {
	db, mock := newTxnMock(t)
	mock.ExpectBegin()
	mock.ExpectRollback().WillReturnError(errors.New("rollback failed"))

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	}()

	NewTxn(db, sqlc.New(db)).Exec(context.Background(), func(_ context.Context, _ *sql.Tx, _ *sqlc.Queries) error {
		panic("boom")
	})
}

func TestExecRetriesOnRetryableError(t *testing.T) {
	db, mock := newTxnMock(t)
	retryableErr := errors.New("Deadlock found when trying to get lock")

	mock.ExpectBegin()
	mock.ExpectRollback()
	mock.ExpectBegin()
	mock.ExpectCommit()

	calls := 0
	err := NewTxn(db, sqlc.New(db), WithRetries(1)).Exec(
		context.Background(),
		func(_ context.Context, _ *sql.Tx, _ *sqlc.Queries) error {
			calls++
			if calls == 1 {
				return retryableErr
			}
			return nil
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("got %d calls, want 2", calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestExecNoRetryOnNonRetryableError(t *testing.T) {
	db, mock := newTxnMock(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	calls := 0
	err := NewTxn(db, sqlc.New(db)).Exec(
		context.Background(),
		func(_ context.Context, _ *sql.Tx, _ *sqlc.Queries) error {
			calls++
			return errors.New("syntax error")
		},
	)

	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("got %d calls, want 1", calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestExecRetriesUpToLimit(t *testing.T) {
	db, mock := newTxnMock(t)
	retryableErr := errors.New("Table definition has changed, please retry transaction")

	for i := 0; i < 4; i++ {
		mock.ExpectBegin()
		mock.ExpectRollback()
	}

	calls := 0
	err := NewTxn(db, sqlc.New(db)).Exec(
		context.Background(),
		func(_ context.Context, _ *sql.Tx, _ *sqlc.Queries) error {
			calls++
			return retryableErr
		},
	)

	if !errors.Is(err, retryableErr) {
		t.Fatalf("got %v, want %v", err, retryableErr)
	}
	if calls != 4 {
		t.Fatalf("got %d calls, want 4", calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestExecNoRetryOnCancelledContext(t *testing.T) {
	db, mock := newTxnMock(t)
	retryableErr := errors.New("Deadlock found when trying to get lock")
	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := NewTxn(db, sqlc.New(db)).Exec(
		ctx,
		func(_ context.Context, _ *sql.Tx, _ *sqlc.Queries) error {
			calls++
			cancel()
			return retryableErr
		},
	)

	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("got %d calls, want 1", calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestExecBackoffDelays(t *testing.T) {
	db, mock := newTxnMock(t)
	retryableErr := errors.New("Lock wait timeout exceeded")

	mock.ExpectBegin()
	mock.ExpectRollback()
	mock.ExpectBegin()
	mock.ExpectCommit()

	start := time.Now()
	calls := 0
	err := NewTxn(db, sqlc.New(db),
		WithRetries(1),
		WithBackoff(func(_ int) time.Duration { return 50 * time.Millisecond }),
	).Exec(
		context.Background(),
		func(_ context.Context, _ *sql.Tx, _ *sqlc.Queries) error {
			calls++
			if calls == 1 {
				return retryableErr
			}
			return nil
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("got %d calls, want 2", calls)
	}
	if elapsed := time.Since(start); elapsed < 50*time.Millisecond {
		t.Fatalf("backoff too short: %v", elapsed)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestExecBackoffCancelledDuringWait(t *testing.T) {
	db, mock := newTxnMock(t)
	retryableErr := errors.New("Wait on a lock was aborted due to a pending exclusive lock")

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := NewTxn(db, sqlc.New(db),
		WithRetries(1),
		WithBackoff(func(_ int) time.Duration { return 5 * time.Second }),
	).Exec(
		ctx,
		func(_ context.Context, _ *sql.Tx, _ *sqlc.Queries) error {
			return retryableErr
		},
	)

	if !errors.Is(err, retryableErr) {
		t.Fatalf("got %v, want %v", err, retryableErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestWithRetryString(t *testing.T) {
	db, mock := newTxnMock(t)

	mock.ExpectBegin()
	mock.ExpectRollback()
	mock.ExpectBegin()
	mock.ExpectCommit()

	calls := 0
	err := NewTxn(db, sqlc.New(db),
		WithRetries(1),
		WithRetryString("custom retryable"),
	).Exec(
		context.Background(),
		func(_ context.Context, _ *sql.Tx, _ *sqlc.Queries) error {
			calls++
			if calls == 1 {
				return errors.New("custom retryable error")
			}
			return nil
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("got %d calls, want 2", calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestWithRetryFunc(t *testing.T) {
	db, mock := newTxnMock(t)

	mock.ExpectBegin()
	mock.ExpectRollback()
	mock.ExpectBegin()
	mock.ExpectCommit()

	sentinel := errors.New("sentinel")
	calls := 0
	err := NewTxn(db, sqlc.New(db),
		WithRetries(1),
		WithRetryFunc(func(err error) bool { return errors.Is(err, sentinel) }),
	).Exec(
		context.Background(),
		func(_ context.Context, _ *sql.Tx, _ *sqlc.Queries) error {
			calls++
			if calls == 1 {
				return fmt.Errorf("wrapped: %w", sentinel)
			}
			return nil
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("got %d calls, want 2", calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDefaultRetryStrings(t *testing.T) {
	txn := NewTxn(nil, nil)

	retryable := []string{
		"Error 1213: Deadlock found when trying to get lock; try restarting",
		"Error 1205: Lock wait timeout exceeded; try restarting",
		"Error 1412: Table definition has changed, please retry transaction",
		"Error 1689: Wait on a lock was aborted due to a pending exclusive lock",
		"Error 3058: Deadlock found when trying to get user-level lock; try rolling back",
	}
	for _, msg := range retryable {
		if !txn.isRetryable(errors.New(msg)) {
			t.Errorf("expected retryable: %s", msg)
		}
	}
	if txn.isRetryable(errors.New("syntax error")) {
		t.Error("expected non-retryable: syntax error")
	}
}

func TestWithRetryFuncsClearsDefaults(t *testing.T) {
	txn := NewTxn(nil, nil, WithRetryFuncs())

	if txn.isRetryable(errors.New("Deadlock found when trying to get lock")) {
		t.Error("expected non-retryable after WithRetryFuncs()")
	}
}

func TestWithRetryFuncsReplacesDefaults(t *testing.T) {
	txn := NewTxn(nil, nil, WithRetryFuncs(func(err error) bool {
		return err.Error() == "custom"
	}))

	if txn.isRetryable(errors.New("Deadlock found when trying to get lock")) {
		t.Error("expected default retry string to be cleared")
	}
	if !txn.isRetryable(errors.New("custom")) {
		t.Error("expected custom retry func to match")
	}
}

func TestExecRetriesExhaustedMessage(t *testing.T) {
	db, mock := newTxnMock(t)
	retryableErr := errors.New("Deadlock found when trying to get lock")

	for i := 0; i <= DefaultRetries; i++ {
		mock.ExpectBegin()
		mock.ExpectRollback()
	}

	err := NewTxn(db, sqlc.New(db)).Exec(
		context.Background(),
		func(_ context.Context, _ *sql.Tx, _ *sqlc.Queries) error {
			return retryableErr
		},
	)

	if !errors.Is(err, retryableErr) {
		t.Fatalf("got %v, want wrapped %v", err, retryableErr)
	}
	if !strings.Contains(err.Error(), "exhausted") {
		t.Fatalf("expected 'exhausted' in error, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestExecTxnConvenience(t *testing.T) {
	db, mock := newTxnMock(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	if err := ExecTxn(context.Background(), db, sqlc.New(db), noop); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestExecCommitErrorNotMaskedByRollback(t *testing.T) {
	db, mock := newTxnMock(t)
	mock.ExpectBegin()
	commitErr := errors.New("commit failed")
	mock.ExpectCommit().WillReturnError(commitErr)

	err := NewTxn(db, sqlc.New(db)).Exec(context.Background(), noop)

	if !errors.Is(err, commitErr) {
		t.Fatalf("got %v, want %v", err, commitErr)
	}
	if !strings.Contains(err.Error(), "tx commit") {
		t.Errorf("expected 'tx commit' in error, got: %v", err)
	}
	if strings.Contains(err.Error(), "rollback") {
		t.Errorf("commit failure must not report a rollback, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestExecRollbackErrorReported(t *testing.T) {
	db, mock := newTxnMock(t)
	mock.ExpectBegin()
	mock.ExpectRollback().WillReturnError(errors.New("rollback failed"))

	fnErr := errors.New("bad query")
	err := NewTxn(db, sqlc.New(db)).Exec(
		context.Background(),
		func(_ context.Context, _ *sql.Tx, _ *sqlc.Queries) error {
			return fnErr
		},
	)

	if !errors.Is(err, fnErr) {
		t.Fatalf("got %v, want wrapped %v", err, fnErr)
	}
	if !strings.Contains(err.Error(), "rollback failed") {
		t.Errorf("expected the rollback failure in error, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// txnOp is a transaction operation performed against [txnMock].
type txnOp int

const (
	opBegin txnOp = iota
	opCommit
	opRollback
)

func (o txnOp) String() string {
	switch o {
	case opBegin:
		return "begin"
	case opCommit:
		return "commit"
	case opRollback:
		return "rollback"
	}
	return "unknown"
}

// txnExpect is a single expected operation and the error it should return.
type txnExpect struct {
	op  txnOp
	err error
}

// WillReturnError makes the expected operation fail with err.
func (e *txnExpect) WillReturnError(err error) *txnExpect {
	e.err = err
	return e
}

// txnMock is a [driver.Connector] that supports no queries but records the
// begin, commit, and rollback operations performed against it, matching them
// against expectations set with ExpectBegin, ExpectCommit, and ExpectRollback.
// It stands in for a database so the transaction tests need no mocking
// dependency and no MySQL server.
type txnMock struct {
	mu     sync.Mutex
	expect []*txnExpect
	pos    int
	err    error // first mismatched or unexpected operation
}

func (m *txnMock) ExpectBegin() *txnExpect    { return m.expectOp(opBegin) }
func (m *txnMock) ExpectCommit() *txnExpect   { return m.expectOp(opCommit) }
func (m *txnMock) ExpectRollback() *txnExpect { return m.expectOp(opRollback) }

func (m *txnMock) expectOp(op txnOp) *txnExpect {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := &txnExpect{op: op}
	m.expect = append(m.expect, e)
	return e
}

// ExpectationsWereMet reports whether every expectation was matched, in order,
// with no extra operations.
func (m *txnMock) ExpectationsWereMet() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	if m.pos < len(m.expect) {
		return fmt.Errorf("%d expectation(s) not met, next: %s", len(m.expect)-m.pos, m.expect[m.pos].op)
	}
	return nil
}

// next matches op against the next expectation and returns the error that
// expectation is configured to return. Mismatches are recorded for
// ExpectationsWereMet rather than returned, so they don't masquerade as
// database errors.
func (m *txnMock) next(op txnOp) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pos >= len(m.expect) {
		m.record(fmt.Errorf("unexpected %s: no expectations left", op))
		return nil
	}
	e := m.expect[m.pos]
	m.pos++
	if e.op != op {
		m.record(fmt.Errorf("expected %s, got %s", e.op, op))
	}
	return e.err
}

// record keeps the first failure. Callers must hold m.mu.
func (m *txnMock) record(err error) {
	if m.err == nil {
		m.err = err
	}
}

func (m *txnMock) Connect(context.Context) (driver.Conn, error) { return &txnMockConn{m: m}, nil }
func (m *txnMock) Driver() driver.Driver                        { return txnMockDriver{m: m} }

type txnMockDriver struct{ m *txnMock }

func (d txnMockDriver) Open(string) (driver.Conn, error) { return &txnMockConn{m: d.m}, nil }

type txnMockConn struct{ m *txnMock }

func (c *txnMockConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("txnMock: queries not supported")
}

func (c *txnMockConn) Close() error { return nil }

func (c *txnMockConn) Begin() (driver.Tx, error) {
	if err := c.m.next(opBegin); err != nil {
		return nil, err
	}
	return &txnMockTx{m: c.m}, nil
}

type txnMockTx struct{ m *txnMock }

func (t *txnMockTx) Commit() error   { return t.m.next(opCommit) }
func (t *txnMockTx) Rollback() error { return t.m.next(opRollback) }
