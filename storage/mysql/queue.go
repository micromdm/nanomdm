package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jessepeterson/nanomdm/mdm"
)

func enqueue(tx *sql.Tx, ctx context.Context, ids []string, cmd *mdm.Command) error {
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
	args := []interface{}{ids[0], cmd.CommandUUID}
	for _, id := range ids[1:] {
		query += `, (?, ?)`
		args = append(args, id, cmd.CommandUUID)
	}
	_, err = tx.ExecContext(ctx, query, args...)
	return err
}

func (m *MySQLStorage) EnqueueCommand(ctx context.Context, ids []string, cmd *mdm.Command) (map[string]error, error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if err = enqueue(tx, ctx, ids, cmd); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return nil, fmt.Errorf("rollback error: %w; while trying to handle error: %v", rbErr, err)
		}
		return nil, err
	}
	return nil, tx.Commit()
}

func (s *MySQLStorage) StoreCommandReport(r *mdm.Request, result *mdm.CommandResults) error {
	if result.Status == "Idle" {
		// TODO: store LastSeen?
		return nil
	}
	exists, err := s.queryRowContextRowExists(
		r.Context,
		`SELECT COUNT(*) FROM command_results WHERE id = ? AND command_uuid = ?`,
		r.ID, result.CommandUUID,
	)
	if err != nil {
		return err
	}
	if exists {
		_, err = s.db.ExecContext(
			r.Context,
			`UPDATE command_results SET status = ?, result = ? WHERE id = ? AND command_uuid = ?;`,
			result.Status,
			result.Raw,
			r.ID,
			result.CommandUUID,
		)
	} else {
		_, err = s.db.ExecContext(
			r.Context,
			`INSERT INTO command_results (id, command_uuid, status, result) VALUES (?, ?, ?, ?);`,
			r.ID,
			result.CommandUUID,
			result.Status,
			result.Raw,
		)
	}
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
