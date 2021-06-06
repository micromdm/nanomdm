// Pacakge mysql stores and retrieves MDM data from SQL
package mysql

import (
	"database/sql"
	"errors"

	"github.com/micromdm/nanomdm/cryptoutil"
	"github.com/micromdm/nanomdm/log"
	"github.com/micromdm/nanomdm/mdm"

	_ "github.com/go-sql-driver/mysql"
)

var ErrNoCert = errors.New("no certificate in MDM Request")

type MySQLStorage struct {
	logger log.Logger
	db     *sql.DB
}

func New(conn string, logger log.Logger) (*MySQLStorage, error) {
	db, err := sql.Open("mysql", conn)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return &MySQLStorage{db: db, logger: logger}, nil
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
	(id, device_id, user_id, type, topic, push_magic, token_hex)
VALUES
	(?, ?, ?, ?, ?, ?, ?) AS new
ON DUPLICATE KEY
UPDATE
    device_id = new.device_id,
    user_id = new.user_id,
    type = new.type,
    topic = new.topic,
    push_magic = new.push_magic,
    token_hex = new.token_hex,
	enabled = 1;`,
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

func (s *MySQLStorage) Disable(r *mdm.Request) error {
	if r.ParentID != "" {
		return errors.New("can only disable a device channel")
	}
	_, err := s.db.ExecContext(
		r.Context,
		`UPDATE enrollments SET enabled = 0 WHERE device_id = ? AND enabled = 1;`,
		r.ID,
	)
	return err
}
