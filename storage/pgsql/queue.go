package pgsql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/micromdm/nanomdm/mdm"
)

func enqueue(ctx context.Context, tx *sql.Tx, ids []string, cmd *mdm.Command) error {
	if len(ids) < 1 {
		return errors.New("no id(s) supplied to queue command to")
	}
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO commands (command_uuid, request_type, command) VALUES ($1, $2, $3);`,
		cmd.CommandUUID, cmd.Command.RequestType, cmd.Raw,
	)
	if err != nil {
		return err
	}

	var query strings.Builder

	query.WriteString(`INSERT INTO enrollment_queue (id, command_uuid) VALUES `)
	args := make([]interface{}, len(ids)*2)
	for i, id := range ids {
		if i > 0 {
			query.WriteString(",")
		}
		ind := i * 2

		//previous: query += fmt.Sprintf("($%d, $%d)", ind+1, ind+2)
		query.WriteString("($")
		query.WriteString(strconv.Itoa(ind + 1))
		query.WriteString(", $")
		query.WriteString(strconv.Itoa(ind + 2))
		query.WriteString(")")

		args[ind] = id
		args[ind+1] = cmd.CommandUUID
	}
	query.WriteString(";")

	_, err = tx.ExecContext(ctx, query.String(), args...)
	return err
}

func (s *PgSQLStorage) EnqueueCommand(ctx context.Context, ids []string, cmd *mdm.Command) (map[string]error, error) {
	tx, err := s.db.BeginTx(ctx, nil)
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

func (s *PgSQLStorage) StoreCommandReport(r *mdm.Request, result *mdm.CommandResults) error {
	if err := s.updateLastSeen(r); err != nil {
		return err
	}
	if result.Status == "Idle" {
		return nil
	}
	notNowConstants := "NULL, 0"
	notNowBumpTallySQL := ""
	// note that due to the "ON CONFLICT ON CONSTRAINT command_results_pkey" we don't UPDATE the
	// not_now_at field. thus it will only represent the first NotNow.
	if result.Status == "NotNow" {
		notNowConstants = "CURRENT_TIMESTAMP, 1"
		notNowBumpTallySQL = `, not_now_tally = command_results.not_now_tally + 1`
	}
	_, err := s.db.ExecContext(
		r.Context, `
INSERT INTO command_results
    (id, command_uuid, status, result, not_now_at, not_now_tally)
VALUES
    ($1, $2, $3, $4, `+notNowConstants+`)
ON CONFLICT ON CONSTRAINT command_results_pkey DO UPDATE 
SET
    status = EXCLUDED.status,
    result = EXCLUDED.result`+notNowBumpTallySQL+`;`,
		r.ID,
		result.CommandUUID,
		result.Status,
		result.Raw,
	)
	return err
}

func (s *PgSQLStorage) RetrieveNextCommand(r *mdm.Request, skipNotNow bool) (*mdm.Command, error) {
	statusWhere := "status IS NULL"
	if !skipNotNow {
		statusWhere = `(` + statusWhere + ` OR status = 'NotNow')`
	}
	command := new(mdm.Command)
	err := s.db.QueryRowContext(
		r.Context,
		`SELECT command_uuid, request_type, command FROM view_queue WHERE id = $1 AND active = TRUE AND `+statusWhere+` LIMIT 1;`,
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

func (s *PgSQLStorage) ClearQueue(r *mdm.Request) error {
	if r.ParentID != "" {
		return errors.New("can only clear a device channel queue")
	}
	// PostgreSQL UPDATE differs from MySQL, uses "FROM" specific
	// to pgsql extension
	_, err := s.db.ExecContext(
		r.Context,
		`
UPDATE enrollment_queue
SET active = FALSE
FROM enrollment_queue  AS q
	INNER JOIN enrollments AS e
		ON q.id = e.id
	INNER JOIN commands AS c
		ON q.command_uuid = c.command_uuid
	LEFT JOIN command_results r
		ON r.command_uuid = q.command_uuid AND r.id = q.id
WHERE 
    e.device_id = $1 AND
    enrollment_queue.active = TRUE AND
    (r.status IS NULL OR r.status = 'NotNow') AND 
    enrollment_queue.id = q.id;`,
		r.ID)
	return err
}
