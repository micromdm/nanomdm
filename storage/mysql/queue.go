package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage/mysql/sqlc"
)

// insertEnrollmentQueue inserts commandUUID in the enrollment queue for ids.
// sqlc doesn't have macro support for multi-row operations like this insert.
// manually assemble the query.
func insertEnrollmentQueue(ctx context.Context, tx *sql.Tx, ids []string, commandUUID string) error {
	query := `-- func: insertEnrollmentQueue :exec
INSERT INTO
    enrollment_queue (id, command_uuid)
VALUES
    (?, ?)`
	query += strings.Repeat(", (?, ?)", len(ids)-1) + ";"
	args := make([]interface{}, len(ids)*2)
	for i, id := range ids {
		args[i*2] = id
		args[i*2+1] = commandUUID
	}

	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

// EnqueueCommand inserts command and enrollment queue records.
func (s *MySQLStorage) EnqueueCommand(ctx context.Context, ids []string, cmd *mdm.Command) (map[string]error, error) {
	if len(ids) < 1 {
		return nil, errors.New("no id(s) supplied to queue command to")
	}

	return nil, s.txn.Exec(ctx, func(ctx context.Context, tx *sql.Tx, qtx *sqlc.Queries) error {
		params := sqlc.InsertCommandsParams{
			CommandUuid: cmd.CommandUUID,
			RequestType: cmd.Command.RequestType,
			Command:     cmd.Raw,
		}
		if err := qtx.InsertCommands(ctx, params); err != nil {
			return fmt.Errorf("inserting command: %w", err)
		}

		if err := insertEnrollmentQueue(ctx, tx, ids, cmd.CommandUUID); err != nil {
			return fmt.Errorf("insert enrollment queue : %w", err)
		}

		return nil
	})
}

// StoreCommandReport upserts the command result for r.ID, or deletes the
// command from the queue when DeleteCommands is enabled and the status is
// not "NotNow". A status of "Idle" is not stored.
func (s *MySQLStorage) StoreCommandReport(r *mdm.Request, result *mdm.CommandResults) error {
	defer s.updateLastSeen(r)

	if result.Status == "Idle" {
		return nil
	}

	return s.txn.Exec(r.Context(), func(ctx context.Context, tx *sql.Tx, qtx *sqlc.Queries) error {
		if s.rm && result.Status != "NotNow" {
			// first, place a record lock on the command so that multiple devices
			// trying to each delete it do not race
			if err := qtx.LockCommandsForDelete(ctx, result.CommandUUID); err != nil {
				return fmt.Errorf("lock commands for delete: %w", err)
			}

			// delete command result (i.e. NotNows) and this queued command
			if err := qtx.DeleteCommandResultForID(ctx, sqlc.DeleteCommandResultForIDParams{
				ID: r.ID, CommandUuid: result.CommandUUID,
			}); err != nil {
				return fmt.Errorf("delete command result for id: %w", err)
			}

			// now delete the actual command if no enrollments have it queued
			// nor are there any results for it.
			if err := qtx.DeleteCommandWhereComplete(ctx, result.CommandUUID); err != nil {
				return fmt.Errorf("delete command where complete: %w", err)
			}

			return nil
		}

		// note that due to the ON DUPLICATE KEY we don't UPDATE the
		// not_now_at field. thus it will only represent the first NotNow.
		if result.Status == "NotNow" {
			params := sqlc.UpsertCommandReportNotNowParams{
				ID:          r.ID,
				CommandUuid: result.CommandUUID,
				Status:      result.Status,
				Result:      result.Raw,
			}
			if err := qtx.UpsertCommandReportNotNow(r.Context(), params); err != nil {
				return fmt.Errorf("upsert command report not now: %w", err)
			}
		} else {
			params := sqlc.UpsertCommandReportParams{
				ID:          r.ID,
				CommandUuid: result.CommandUUID,
				Status:      result.Status,
				Result:      result.Raw,
			}
			if err := qtx.UpsertCommandReport(r.Context(), params); err != nil {
				return fmt.Errorf("upsert command report: %w", err)
			}
		}
		return nil
	})
}

// RetrieveNextCommand selects the next command in the queue for r.ID.
// If skipNotNow is true then commands with a "NotNow" result are skipped.
// A nil command is returned when the queue is empty.
func (s *MySQLStorage) RetrieveNextCommand(r *mdm.Request, skipNotNow bool) (*mdm.Command, error) {
	var includeNotNow int64
	if !skipNotNow {
		includeNotNow = 1
	}
	dbCommand, err := s.q.SelectNextCommand(
		r.Context(),
		sqlc.SelectNextCommandParams{
			ID:            r.ID,
			IncludeNotNow: includeNotNow, // inverse of skipNotNow; CAST to SIGNED so sqlc types it as int64
		},
	)
	if errors.Is(err, sql.ErrNoRows) {
		// it's valid to have nothing in a device's command queue
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	command := &mdm.Command{
		CommandUUID: dbCommand.CommandUuid,
		Command: struct{ RequestType string }{
			RequestType: dbCommand.RequestType,
		},
		Raw: dbCommand.Command,
	}

	return command, nil
}

// ClearQueue updates (marks inactive) the queued commands for r.ID's
// device channel, including user-channel enrollments with r.ID as parent.
func (s *MySQLStorage) ClearQueue(r *mdm.Request) error {
	if r.ParentID != "" {
		return errors.New("can only clear a device channel queue")
	}
	// Because we're joining on and WHERE-ing by the enrollments table
	// this will clear (mark inactive) the queue of not only this
	// device ID, but all user-channel enrollments with a 'parent' ID of
	// this device, too.
	return s.q.UpdateClearQueue(r.Context(), r.ID)
}
