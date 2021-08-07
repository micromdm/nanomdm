// Pacakge mysql stores and retrieves MDM data from SQL
package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/micromdm/nanomdm/cryptoutil"
	"github.com/micromdm/nanomdm/log"
	"github.com/micromdm/nanomdm/mdm"
)

var ErrNoCert = errors.New("no certificate in MDM Request")

type MySQLStorage struct {
	logger log.Logger
	db     *sql.DB
}

type config struct {
	driver string
	dsn    string
	db     *sql.DB
	logger log.Logger
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

func New(opts ...Option) (*MySQLStorage, error) {
	cfg := &config{logger: log.NopLogger, driver: "mysql"}
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
	if err = cfg.db.Ping(); err != nil {
		return nil, err
	}
	return &MySQLStorage{db: cfg.db, logger: cfg.logger}, nil
}

// nullEmptyString returns a NULL string if s is empty.
func nullEmptyString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}

func (s *MySQLStorage) StoreAuthenticate(r *mdm.Request, msg *mdm.Authenticate) error {
	var pemCert []byte
	if r.Certificate != nil {
		pemCert = cryptoutil.PEMCertificate(r.Certificate.Raw)
	}
	_, err := s.db.ExecContext(
		r.Context, `
INSERT INTO devices
    (id, identity_cert, serial_number, authenticate, authenticate_at)
VALUES
    (?, ?, ?, ?, CURRENT_TIMESTAMP) AS new
ON DUPLICATE KEY
UPDATE
    identity_cert = new.identity_cert,
    serial_number = new.serial_number,
    authenticate = new.authenticate,
    authenticate_at = CURRENT_TIMESTAMP;`,
		r.ID, pemCert, nullEmptyString(msg.SerialNumber), msg.Raw,
	)
	return err
}

func (s *MySQLStorage) storeDeviceTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	query := `UPDATE devices SET token_update = ?, token_update_at = CURRENT_TIMESTAMP`
	args := []interface{}{msg.Raw}
	// separately store the Unlock Token per MDM spec
	if len(msg.UnlockToken) > 0 {
		query += `, unlock_token = ?, unlock_token_at = CURRENT_TIMESTAMP`
		args = append(args, msg.UnlockToken)
	}
	query += ` WHERE id = ? LIMIT 1;`
	args = append(args, r.ID)
	_, err := s.db.ExecContext(r.Context, query, args...)
	return err
}

func (s *MySQLStorage) storeUserTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	// there shouldn't be an Unlock Token on the user channel, but
	// complain if there is to warn an admin
	if len(msg.UnlockToken) > 0 {
		s.logger.Info("msg", "Unlock Token on user channel not stored")
	}
	_, err := s.db.ExecContext(
		r.Context, `
INSERT INTO users
    (id, device_id, user_short_name, user_long_name, token_update, token_update_at)
VALUES
    (?, ?, ?, ?, ?, CURRENT_TIMESTAMP) AS new
ON DUPLICATE KEY
UPDATE
    device_id = new.device_id,
    user_short_name = new.user_short_name,
    user_long_name = new.user_long_name,
    token_update = new.token_update,
    token_update_at = CURRENT_TIMESTAMP;`,
		r.ID,
		r.ParentID,
		nullEmptyString(msg.UserShortName),
		nullEmptyString(msg.UserLongName),
		msg.Raw,
	)
	return err
}

func (s *MySQLStorage) StoreTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	var err error
	var deviceId, userId string
	resolved := (&msg.Enrollment).Resolved()
	if err = resolved.Validate(); err != nil {
		return err
	}
	if resolved.IsUserChannel {
		deviceId = r.ParentID
		userId = r.ID
		err = s.storeUserTokenUpdate(r, msg)
	} else {
		deviceId = r.ID
		err = s.storeDeviceTokenUpdate(r, msg)
	}
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(
		r.Context, `
INSERT INTO enrollments
	(id, device_id, user_id, type, topic, push_magic, token_hex, token_update_tally)
VALUES
	(?, ?, ?, ?, ?, ?, ?, 1) AS new
ON DUPLICATE KEY
UPDATE
    device_id = new.device_id,
    user_id = new.user_id,
    type = new.type,
    topic = new.topic,
    push_magic = new.push_magic,
    token_hex = new.token_hex,
	enabled = 1,
	enrollments.token_update_tally = enrollments.token_update_tally + 1;`,
		r.ID,
		deviceId,
		nullEmptyString(userId),
		r.Type.String(),
		msg.Topic,
		msg.PushMagic,
		msg.Token.String(),
	)
	return err
}

func (s *MySQLStorage) RetrieveTokenUpdateTally(ctx context.Context, id string) (int, error) {
	var tally int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT token_update_tally FROM enrollments WHERE id = ?;`,
		id,
	).Scan(&tally)
	return tally, err
}

func (s *MySQLStorage) StoreUserAuthenticate(r *mdm.Request, msg *mdm.UserAuthenticate) error {
	colName := "user_authenticate"
	colAtName := "user_authenticate_at"
	// if the DigestResponse is empty then this is the first (of two)
	// UserAuthenticate messages depending on our response
	if msg.DigestResponse != "" {
		colName = "user_authenticate_digest"
		colAtName = "user_authenticate_digest_at"
	}
	_, err := s.db.ExecContext(
		r.Context, `
INSERT INTO users
    (id, device_id, user_short_name, user_long_name, `+colName+`, `+colAtName+`)
VALUES
    (?, ?, ?, ?, ?, CURRENT_TIMESTAMP) AS new
ON DUPLICATE KEY
UPDATE
    device_id = new.device_id,
    user_short_name = new.user_short_name,
    user_long_name = new.user_long_name,
    `+colName+` = new.`+colName+`,
    `+colAtName+` = new.`+colAtName+`;`,
		r.ID,
		r.ParentID,
		nullEmptyString(msg.UserShortName),
		nullEmptyString(msg.UserLongName),
		msg.Raw,
	)
	return err
}

func (s *MySQLStorage) Disable(r *mdm.Request) error {
	if r.ParentID != "" {
		return errors.New("can only disable a device channel")
	}
	_, err := s.db.ExecContext(
		r.Context,
		`UPDATE enrollments SET enabled = 0, token_update_tally = 0 WHERE device_id = ? AND enabled = 1;`,
		r.ID,
	)
	return err
}
