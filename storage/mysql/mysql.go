// Package mysql stores and retrieves MDM data from MySQL
package mysql

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage/mysql/sqlc"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

// Connection pool recycling is left at database/sql's defaults unless
// configured. Pooled connections should be recycled well before the shortest
// idle timeout in the network path (the istio-proxy sidecar reaps idle TCP
// connections at ~1h, Aurora's wait_timeout is longer). Without recycling,
// database/sql hands out long-idle connections that the far end has already
// closed, producing "broken pipe" writes and "closing bad idle connection:
// EOF" errors. Set WithConnMaxLifetime / WithConnMaxIdleTime per-deployment to
// enable recycling for the path's idle timeout.

// Schema holds the schema for the NanoMDM MySQL storage.
//
//go:embed schema.sql
var Schema string

var ErrNoCert = errors.New("no certificate in MDM Request")

type MySQLStorage struct {
	logger log.Logger
	db     *sql.DB
	q      *sqlc.Queries
	rm     bool
}

type config struct {
	driver          string
	dsn             string
	db              *sql.DB
	logger          log.Logger
	rm              bool
	connMaxLifetime time.Duration
	connMaxIdleTime time.Duration
}

type Option func(*config)

func WithLogger(logger log.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

func WithDSN(dsn string) Option {
	return func(c *config) {
		c.dsn = dsn
	}
}

func WithDriver(driver string) Option {
	return func(c *config) {
		c.driver = driver
	}
}

func WithDB(db *sql.DB) Option {
	return func(c *config) {
		c.db = db
	}
}

func WithDeleteCommands() Option {
	return func(c *config) {
		c.rm = true
	}
}

// WithConnMaxLifetime sets the maximum amount of time a connection may be
// reused. It should be shorter than the shortest idle timeout in the network
// path to the database. A non-positive value keeps connections forever.
func WithConnMaxLifetime(d time.Duration) Option {
	return func(c *config) {
		c.connMaxLifetime = d
	}
}

// WithConnMaxIdleTime sets the maximum amount of time a connection may be idle
// before it is closed. A non-positive value never closes connections due to
// idle time.
func WithConnMaxIdleTime(d time.Duration) Option {
	return func(c *config) {
		c.connMaxIdleTime = d
	}
}

func New(opts ...Option) (*MySQLStorage, error) {
	cfg := &config{
		logger: log.NopLogger,
		driver: "mysql",
	}
	for _, opt := range opts {
		opt(cfg)
	}
	var err error
	if cfg.db == nil {
		cfg.db, err = sql.Open(cfg.driver, cfg.dsn)
		if err != nil {
			return nil, err
		}
	}
	if cfg.connMaxLifetime != 0 {
		cfg.db.SetConnMaxLifetime(cfg.connMaxLifetime)
	}
	if cfg.connMaxIdleTime != 0 {
		cfg.db.SetConnMaxIdleTime(cfg.connMaxIdleTime)
	}
	if err = cfg.db.Ping(); err != nil {
		return nil, err
	}
	return &MySQLStorage{db: cfg.db, q: sqlc.New(cfg.db), logger: cfg.logger, rm: cfg.rm}, nil
}

// nullEmptyString returns a NULL string if s is empty.
func nullEmptyString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}

// txcb executes SQL within transactions when wrapped in tx().
type txcb func(ctx context.Context, tx *sql.Tx, qtx *sqlc.Queries) error

// tx wraps g in transactions using db.
// If g returns an err the transaction will be rolled back; otherwise committed.
func tx(ctx context.Context, db *sql.DB, q *sqlc.Queries, g txcb) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("tx begin: %w", err)
	}
	if err = g(ctx, tx, q.WithTx(tx)); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx rollback: %w; while trying to handle error: %v", rbErr, err)
		}
		return fmt.Errorf("tx rolled back: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}
	return nil
}

// updateLastSeen updates the last seen timestamp for r.ID.
// Typically called from a defer, errors are logged to s.logger.
func (s *MySQLStorage) updateLastSeen(r *mdm.Request) {
	err := s.q.UpdateEnrollmentLastSeen(r.Context(), r.ID)
	if err != nil {
		ctxlog.Logger(r.Context(), s.logger).Info(
			"msg", "updating last seen",
			"id", r.ID,
			"err", err,
		)
	}
}
