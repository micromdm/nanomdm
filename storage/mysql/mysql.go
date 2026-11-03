// Package mysql stores and retrieves MDM data from MySQL
package mysql

import (
	"database/sql"
	_ "embed"
	"errors"
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
//
// The pool's size is likewise unbounded unless configured. Set
// WithMaxOpenConns to bound it so that a burst of concurrent requests queues
// on the pool rather than opening connections until the database refuses them.

// Schema holds the schema for the NanoMDM MySQL storage.
//
//go:embed schema.sql
var Schema string

var ErrNoCert = errors.New("no certificate in MDM Request")

type MySQLStorage struct {
	logger log.Logger
	db     *sql.DB
	q      *sqlc.Queries
	txn    *Txn
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
	maxOpenConns    int
	txnOpts         []TxnOption
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

// WithMaxOpenConns sets the maximum number of open connections to the
// database. A non-positive value leaves database/sql's default of no limit,
// under which a burst of concurrent requests opens a connection per in-flight
// request until the server refuses them. Bounding the pool turns that into
// callers waiting on a connection, which is visible as the wait count and wait
// duration in [database/sql.DB.Stats].
func WithMaxOpenConns(n int) Option {
	return func(c *config) {
		c.maxOpenConns = n
	}
}

// WithTxnOptions configures the transactions this storage runs, overriding
// the defaults documented on [NewTxn]. Options accumulate across calls.
func WithTxnOptions(opts ...TxnOption) Option {
	return func(c *config) {
		c.txnOpts = append(c.txnOpts, opts...)
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
	if cfg.maxOpenConns > 0 {
		cfg.db.SetMaxOpenConns(cfg.maxOpenConns)
	}
	if err = cfg.db.Ping(); err != nil {
		return nil, err
	}
	q := sqlc.New(cfg.db)
	return &MySQLStorage{
		db:     cfg.db,
		q:      q,
		txn:    NewTxn(cfg.db, q, cfg.txnOpts...),
		logger: cfg.logger,
		rm:     cfg.rm,
	}, nil
}

// nullEmptyString returns a NULL string if s is empty.
func nullEmptyString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
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
