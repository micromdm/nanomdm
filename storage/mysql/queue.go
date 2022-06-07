package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/micromdm/nanomdm/mdm"
)

func enqueue(ctx context.Context, tx *sql.Tx, ids []string, cmd *mdm.Command) error {
	if len(ids) < 1 {
		return errors.New("no id(s) supplied to queue command to")
	}
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO commands (command_uuid, request_type, command) VALUES (?, ?, ?);`,
		cmd.CommandUUID, cmd.Command.RequestType, cmd.Raw,
	)
	if err != nil {
		return err
	}
	query := `INSERT INTO enrollment_queue (id, command_uuid) VALUES (?, ?)`
	query += strings.Repeat(", (?, ?)", len(ids)-1)
	args := make([]interface{}, len(ids)*2)
	for i, id := range ids {
		args[i*2] = id
		args[i*2+1] = cmd.CommandUUID
	}
	_, err = tx.ExecContext(ctx, query+";", args...)
	return err
}

func (m *MySQLStorage) EnqueueCommand(ctx context.Context, ids []string, cmd *mdm.Command) (map[string]error, error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if err = enqueue(ctx, tx, ids, cmd); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return nil, fmt.Errorf("rollback error: %w; while trying to handle error: %v", rbErr, err)
		}
		return nil, err
	}
	return nil, tx.Commit()
}

func (s *MySQLStorage) deleteCommand(ctx context.Context, tx *sql.Tx, id, uuid string) error {
	// delete command result (i.e. NotNows) and this queued command
	_, err := tx.ExecContext(
		ctx, `
DELETE
    q, r
FROM
    enrollment_queue AS q
    LEFT JOIN command_results AS r
        ON q.command_uuid = r.command_uuid AND r.id = q.id
WHERE
    q.id = ? AND q.command_uuid = ?;
`,
		id, uuid,
	)
	if err != nil {
		return err
	}
	// now delete the actual command if no enrollments have it queued
	// nor are there any results for it.
	_, err = tx.ExecContext(
		ctx, `
DELETE
    c
FROM
    commands AS c
    LEFT JOIN enrollment_queue AS q
        ON q.command_uuid = c.command_uuid
    LEFT JOIN command_results AS r
        ON r.command_uuid = c.command_uuid
WHERE
    c.command_uuid = ? AND
    q.command_uuid IS NULL AND
    r.command_uuid IS NULL;
`,
		uuid,
	)
	return err
}

func (s *MySQLStorage) deleteCommandTx(r *mdm.Request, result *mdm.CommandResults) error {
	tx, err := s.db.BeginTx(r.Context, nil)
	if err != nil {
		return err
	}
	if err = s.deleteCommand(r.Context, tx, r.ID, result.CommandUUID); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback error: %w; while trying to handle error: %v", rbErr, err)
		}
		return err
	}
	return tx.Commit()
}

func (s *MySQLStorage) StoreCommandReport(r *mdm.Request, result *mdm.CommandResults) error {
	if err := s.updateLastSeen(r); err != nil {
		return err
	}
	if result.Status == "Idle" {
		return nil
	}
	if s.rm && result.Status != "NotNow" {
		return s.deleteCommandTx(r, result)
	}
	notNowConstants := "NULL, 0"
	notNowBumpTallySQL := ""
	// note that due to the ON DUPLICATE KEY we don't UPDATE the
	// not_now_at field. thus it will only represent the first NotNow.
	if result.Status == "NotNow" {
		notNowConstants = "CURRENT_TIMESTAMP, 1"
		notNowBumpTallySQL = `, command_results.not_now_tally = command_results.not_now_tally + 1`
	}
	_, err := s.db.ExecContext(
		r.Context, `
INSERT INTO command_results
    (id, command_uuid, status, result, not_now_at, not_now_tally)
VALUES
    (?, ?, ?, ?, `+notNowConstants+`) AS new
ON DUPLICATE KEY
UPDATE
    status = new.status,
    result = new.result`+notNowBumpTallySQL+`;`,
		r.ID,
		result.CommandUUID,
		result.Status,
		result.Raw,
	)
	return err
}

func (s *MySQLStorage) RetrieveNextCommand(r *mdm.Request, skipNotNow bool) (*mdm.Command, error) {
	statusWhere := "status IS NULL"
	if !skipNotNow {
		statusWhere = `(` + statusWhere + ` OR status = 'NotNow')`
	}
	command := new(mdm.Command)
	err := s.db.QueryRowContext(
		r.Context,
		`SELECT command_uuid, request_type, command FROM view_queue WHERE id = ? AND active = 1 AND `+statusWhere+` LIMIT 1;`,
		r.ID,
	).Scan(&command.CommandUUID, &command.Command.RequestType, &command.Raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return command, nil
}

func (s *MySQLStorage) ClearQueue(r *mdm.Request) error {
	if r.ParentID != "" {
		return errors.New("can only clear a device channel queue")
	}
	// Because we're joining on and WHERE-ing by the enrollments table
	// this will clear (mark inactive) the queue of not only this
	// device ID, but all user-channel enrollments with a 'parent' ID of
	// this device, too.
	_, err := s.db.ExecContext(
		r.Context,
		`
UPDATE
    enrollment_queue AS q
	INNER JOIN enrollments AS e
	    ON q.id = e.id
    INNER JOIN commands AS c
        ON q.command_uuid = c.command_uuid
    LEFT JOIN command_results r
        ON r.command_uuid = q.command_uuid AND r.id = q.id
SET
    q.active = 0
WHERE
    e.device_id = ? AND
    active = 1 AND
    (r.status IS NULL OR r.status = 'NotNow');`,
		r.ID,
	)
	return err
}
