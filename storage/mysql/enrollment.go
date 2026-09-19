package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/micromdm/nanomdm/storage"
)

func scanEnrollment(row interface{ Scan(...any) error }) (*storage.Enrollment, error) {
	var item storage.Enrollment
	var userID, serial, created, updated sql.NullString
	err := row.Scan(&item.ID, &item.DeviceID, &userID, &item.Type, &serial, &item.Enabled, &item.TokenUpdateTally, &item.LastSeenAt, &created, &updated)
	item.UserID, item.SerialNumber = userID.String, serial.String
	item.CreatedAt, item.UpdatedAt = created.String, updated.String
	return &item, err
}

func (s *MySQLStorage) ListEnrollments(ctx context.Context, query string, limit int) ([]storage.Enrollment, error) {
	const base = `SELECT e.id, e.device_id, e.user_id, e.type, d.serial_number, e.enabled, e.token_update_tally, e.last_seen_at, e.created_at, e.updated_at
		FROM enrollments e LEFT JOIN devices d ON d.id = e.device_id`
	args := []interface{}{}
	sqlQuery := base
	if query != "" {
		sqlQuery += ` WHERE e.id LIKE ? OR e.device_id LIKE ? OR e.user_id LIKE ? OR d.serial_number LIKE ?`
		like := "%" + query + "%"
		args = append(args, like, like, like, like)
	}
	sqlQuery += ` ORDER BY e.last_seen_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query enrollments: %w", err)
	}
	defer rows.Close()
	var items []storage.Enrollment
	for rows.Next() {
		item, err := scanEnrollment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan enrollment: %w", err)
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (s *MySQLStorage) GetEnrollment(ctx context.Context, id string) (*storage.Enrollment, error) {
	row := s.db.QueryRowContext(ctx, `SELECT e.id, e.device_id, e.user_id, e.type, d.serial_number, e.enabled, e.token_update_tally, e.last_seen_at, e.created_at, e.updated_at
		FROM enrollments e LEFT JOIN devices d ON d.id = e.device_id WHERE e.id = ?`, id)
	return scanEnrollment(row)
}
