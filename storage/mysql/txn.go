package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/micromdm/nanomdm/storage/mysql/sqlc"
)

const (
	// Error 1213 (40001): Deadlock found when trying to get lock; try restarting transaction
	ErrorSubStringDeadlock = "Deadlock found when trying to get lock"

	// Error 1205 (HY000): Lock wait timeout exceeded; try restarting transaction
	ErrorSubStringLockWaitTimeout = "Lock wait timeout exceeded"

	// Error 1412 (HY000): Table definition has changed, please retry transaction
	ErrorSubStringTableDefChanged = "Table definition has changed"

	// Error 1689 (HY000): Wait on a lock was aborted due to a pending exclusive lock
	ErrorSubStringLockAborted = "Wait on a lock was aborted due to a pending exclusive lock"

	// Error 3058 (HY000): Deadlock found when trying to get user-level lock; try rolling back transaction/releasing locks and restarting lock acquisition.
	ErrorSubStringUserLockDeadlock = "Deadlock found when trying to get user-level lock"

	// Other errors that may warrant retry depending on the environment (but currently don't):
	//   1040 (ER_CON_COUNT_ERROR): Too many connections
	//   1317 (ER_QUERY_INTERRUPTED): Query execution was interrupted
	//   3024 (ER_QUERY_TIMEOUT): Query execution was interrupted, maximum statement execution time exceeded
	//   3572 (ER_LOCK_NOWAIT): Statement aborted because lock(s) could not be acquired immediately and NOWAIT is set.

	// DefaultBackoff is the base delay for exponential backoff between retries.
	DefaultBackoff = 100 * time.Millisecond

	// DefaultRetries is the default number of retry attempts.
	DefaultRetries = 3
)

// TxFunc executes SQL within a transaction.
type TxFunc func(ctx context.Context, tx *sql.Tx, qtx *sqlc.Queries) error

// RetryFunc reports whether an error is retryable.
type RetryFunc func(error) bool

// TxnOption configures a Txn.
type TxnOption func(*Txn)

// Txn executes functions within a database transaction with panic recovery
// and optional retry and backoff support.
type Txn struct {
	db       *sql.DB
	q        *sqlc.Queries
	retries  int
	retryFns []RetryFunc
	backoff  func(attempt int) time.Duration
}

// NewTxn creates a [Txn] with the given options. Defaults are
// [DefaultRetries], [InstantThenExponentialBackoff] (with [DefaultBackoff]),
// and [WithDefaultRetryStrings]. Use opts to override.
func NewTxn(db *sql.DB, q *sqlc.Queries, opts ...TxnOption) *Txn {
	t := &Txn{
		db:      db,
		q:       q,
		retries: DefaultRetries,
		backoff: InstantThenExponentialBackoff(DefaultBackoff),
	}
	WithDefaultRetryStrings()(t) // callers can override with WithRetryFuncs
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// ExecTxn is a convenience wrapper around NewTxn(...).Exec(...).
func ExecTxn(ctx context.Context, db *sql.DB, q *sqlc.Queries, fn TxFunc, opts ...TxnOption) error {
	return NewTxn(db, q, opts...).Exec(ctx, fn)
}

// WithRetries sets the number of additional attempts after the first failure.
// Negative values are treated as zero.
func WithRetries(n int) TxnOption {
	return func(t *Txn) {
		if n < 0 {
			n = 0
		}
		t.retries = n
	}
}

// WithRetryString adds a substring match for retryable error detection.
func WithRetryString(s string) TxnOption {
	return WithRetryFunc(func(err error) bool {
		return strings.Contains(err.Error(), s)
	})
}

// WithRetryFunc adds a custom function for retryable error detection.
func WithRetryFunc(fn RetryFunc) TxnOption {
	return func(t *Txn) {
		t.retryFns = append(t.retryFns, fn)
	}
}

// WithRetryFuncs replaces all retry functions. Use to clear defaults or
// specify an exact set.
func WithRetryFuncs(fns ...RetryFunc) TxnOption {
	return func(t *Txn) {
		t.retryFns = fns
	}
}

// WithDefaultRetryStrings adds substring matches for known retryable MySQL
// errors: [ErrorSubStringDeadlock], [ErrorSubStringLockWaitTimeout],
// [ErrorSubStringTableDefChanged], [ErrorSubStringLockAborted],
// [ErrorSubStringUserLockDeadlock]. Deployments running a MySQL variant with
// additional retryable errors can add them with [WithRetryString].
func WithDefaultRetryStrings() TxnOption {
	return func(t *Txn) {
		WithRetryString(ErrorSubStringDeadlock)(t)
		WithRetryString(ErrorSubStringLockWaitTimeout)(t)
		WithRetryString(ErrorSubStringTableDefChanged)(t)
		WithRetryString(ErrorSubStringLockAborted)(t)
		WithRetryString(ErrorSubStringUserLockDeadlock)(t)
	}
}

// WithBackoff sets a delay function called between retry attempts.
// The function receives the zero-based attempt number of the attempt that
// just failed and returns how long to wait before the next attempt.
func WithBackoff(fn func(attempt int) time.Duration) TxnOption {
	return func(t *Txn) {
		t.backoff = fn
	}
}

// ExponentialBackoff returns a backoff function that doubles the base delay
// each attempt: base, 2*base, 4*base, ...
func ExponentialBackoff(base time.Duration) func(int) time.Duration {
	return func(attempt int) time.Duration {
		return base << attempt
	}
}

// InstantThenExponentialBackoff returns a backoff function that retries
// immediately on the first attempt, then applies exponential backoff:
// 0, base, 2*base, 4*base, ...
func InstantThenExponentialBackoff(base time.Duration) func(int) time.Duration {
	return func(attempt int) time.Duration {
		if attempt == 0 {
			return 0
		}
		return base << (attempt - 1)
	}
}

func (t *Txn) isRetryable(err error) bool {
	for _, fn := range t.retryFns {
		if fn(err) {
			return true
		}
	}
	return false
}

// Exec runs fn within a transaction. On success the transaction is committed.
// On failure it is rolled back and, if the error is retryable, the entire
// transaction is re-attempted up to retries additional times. Panics from
// fn are never retried: the transaction is rolled back and the panic is
// re-propagated.
func (t *Txn) Exec(ctx context.Context, fn TxFunc) error {
	var lastErr error
	for attempt := 0; attempt <= t.retries; attempt++ {
		err := t.execOne(ctx, fn)
		if err == nil {
			return nil
		}
		lastErr = err
		if !t.isRetryable(err) || ctx.Err() != nil {
			return err
		}
		if attempt >= t.retries {
			return fmt.Errorf("tx retry: exhausted %d attempts: %w", t.retries, err)
		}
		if t.backoff != nil {
			if d := t.backoff(attempt); d > 0 {
				select {
				case <-time.After(d):
				case <-ctx.Done():
					return lastErr
				}
			}
		}
	}
	return lastErr
}

func (t *Txn) execOne(ctx context.Context, fn TxFunc) (err error) {
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("tx begin: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			// Intentionally discarding any Rollback error here.
			// We re-panic with the original value so that upstream
			// recover() callers see the original type and value
			// unmodified. A rollback failing during a panic seems
			// like a rare edge case and not worth losing
			// the original panic value over.
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			// ErrTxDone is not a failure. Either the commit already ended
			// the transaction, or database/sql's context-cancellation
			// goroutine won the race to roll back — in which case the
			// rollback did happen, just not from here.
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				err = fmt.Errorf("tx rollback: %v; while handling: %w", rbErr, err)
			}
		}
	}()
	if err = fn(ctx, tx, t.q.WithTx(tx)); err != nil {
		return fmt.Errorf("tx: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}
	return nil
}
