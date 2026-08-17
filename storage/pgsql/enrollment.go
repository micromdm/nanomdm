package pgsql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
)

const defaultEnrollmentsLimit = 100

// QueryEnrollments queries enrollments based on the provided request.
func (s *PgSQLStorage) QueryEnrollments(ctx context.Context, req *storage.EnrollmentsQuery) (*storage.EnrollmentsQueryResult, error) {
	// Build the query
	query, args := s.buildEnrollmentsQuery(req)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying enrollments: %w", err)
	}
	defer rows.Close()

	result := &storage.EnrollmentsQueryResult{
		Enrollments: make([]*storage.Enrollment, 0),
	}

	for rows.Next() {
		enrollment, err := s.scanEnrollment(rows, req)
		if err != nil {
			return nil, fmt.Errorf("scanning enrollment: %w", err)
		}
		result.Enrollments = append(result.Enrollments, enrollment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating enrollments: %w", err)
	}

	return result, nil
}

// buildEnrollmentsQuery builds the SQL query and arguments for enrollments query.
func (s *PgSQLStorage) buildEnrollmentsQuery(req *storage.EnrollmentsQuery) (string, []interface{}) {
	var args []interface{}
	var conditions []string
	argNum := 1

	// Determine which columns to select
	selectCols := `
		e.id,
		e.type,
		e.enabled,
		e.token_update_tally,
		e.last_seen_at,
		d.serial_number,
		u.user_short_name,
		u.user_long_name`

	// Add optional columns based on options
	includeDeviceCert := false
	includeUnlockToken := false
	if req != nil && req.Options != nil {
		includeDeviceCert = req.Options.IncludeDeviceCert
		includeUnlockToken = req.Options.IncludeUnlockToken
	}

	if includeDeviceCert {
		selectCols += `, d.identity_cert`
	}
	if includeUnlockToken {
		selectCols += `, d.unlock_token`
	}

	// Build FROM and JOINs
	fromClause := `
		FROM enrollments e
		INNER JOIN devices d ON e.device_id = d.id
		LEFT JOIN users u ON e.user_id = u.id`

	// Build WHERE conditions based on filter
	if req != nil && req.Filter != nil {
		f := req.Filter

		if len(f.IDs) > 0 {
			placeholders := make([]string, len(f.IDs))
			for i, id := range f.IDs {
				placeholders[i] = fmt.Sprintf("$%d", argNum)
				args = append(args, id)
				argNum++
			}
			conditions = append(conditions, fmt.Sprintf("e.id IN (%s)", strings.Join(placeholders, ", ")))
		}

		if len(f.Serials) > 0 {
			placeholders := make([]string, len(f.Serials))
			for i, serial := range f.Serials {
				placeholders[i] = fmt.Sprintf("$%d", argNum)
				args = append(args, serial)
				argNum++
			}
			conditions = append(conditions, fmt.Sprintf("d.serial_number IN (%s)", strings.Join(placeholders, ", ")))
		}

		if len(f.UserShortNames) > 0 {
			placeholders := make([]string, len(f.UserShortNames))
			for i, name := range f.UserShortNames {
				placeholders[i] = fmt.Sprintf("$%d", argNum)
				args = append(args, name)
				argNum++
			}
			conditions = append(conditions, fmt.Sprintf("u.user_short_name IN (%s)", strings.Join(placeholders, ", ")))
		}

		if len(f.Types) > 0 {
			placeholders := make([]string, len(f.Types))
			for i, t := range f.Types {
				placeholders[i] = fmt.Sprintf("$%d", argNum)
				args = append(args, t)
				argNum++
			}
			conditions = append(conditions, fmt.Sprintf("e.type IN (%s)", strings.Join(placeholders, ", ")))
		}

		if f.Enabled != nil {
			conditions = append(conditions, fmt.Sprintf("e.enabled = $%d", argNum))
			args = append(args, *f.Enabled)
			argNum++
		}
	}

	// Build WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Build ORDER BY and LIMIT
	orderClause := " ORDER BY e.id ASC"

	// Handle pagination
	offset, limit := 0, defaultEnrollmentsLimit
	if req != nil && req.Pagination != nil {
		offset, limit = req.Pagination.DefaultOffsetLimit(defaultEnrollmentsLimit)
	}

	limitClause := fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	query := "SELECT" + selectCols + fromClause + whereClause + orderClause + limitClause

	return query, args
}

// scanEnrollment scans a single enrollment row.
func (s *PgSQLStorage) scanEnrollment(rows *sql.Rows, req *storage.EnrollmentsQuery) (*storage.Enrollment, error) {
	var (
		id               string
		enrollType       string
		enabled          bool
		tokenUpdateTally int
		lastSeen         sql.NullTime
		serialNumber     sql.NullString
		userShortName    sql.NullString
		userLongName     sql.NullString
		identityCert     sql.NullString
		unlockToken      []byte
	)

	// Determine which columns to scan based on options
	includeDeviceCert := false
	includeUnlockToken := false
	if req != nil && req.Options != nil {
		includeDeviceCert = req.Options.IncludeDeviceCert
		includeUnlockToken = req.Options.IncludeUnlockToken
	}

	// Build scan destinations
	scanDest := []interface{}{
		&id,
		&enrollType,
		&enabled,
		&tokenUpdateTally,
		&lastSeen,
		&serialNumber,
		&userShortName,
		&userLongName,
	}

	if includeDeviceCert {
		scanDest = append(scanDest, &identityCert)
	}
	if includeUnlockToken {
		scanDest = append(scanDest, &unlockToken)
	}

	if err := rows.Scan(scanDest...); err != nil {
		return nil, err
	}

	enrollment := &storage.Enrollment{
		ID:               id,
		Type:             parseEnrollType(enrollType),
		Enabled:          enabled,
		TokenUpdateTally: tokenUpdateTally,
	}

	if lastSeen.Valid {
		enrollment.LastSeen = lastSeen.Time
	}

	// Always include device data
	enrollment.Device = &storage.DeviceEnrollment{}
	if serialNumber.Valid {
		enrollment.Device.SerialNumber = serialNumber.String
	}
	if includeDeviceCert && identityCert.Valid {
		enrollment.Device.DeviceCert = identityCert.String
	}
	if includeUnlockToken && len(unlockToken) > 0 {
		enrollment.Device.UnlockToken = unlockToken
	}

	// Include user data if present
	if userShortName.Valid || userLongName.Valid {
		enrollment.User = &storage.UserEnrollment{}
		if userShortName.Valid {
			enrollment.User.UserShortName = userShortName.String
		}
		if userLongName.Valid {
			enrollment.User.UserLongName = userLongName.String
		}
	}

	return enrollment, nil
}

// parseEnrollType converts a string enrollment type to mdm.EnrollType.
func parseEnrollType(s string) mdm.EnrollType {
	switch s {
	case "Device":
		return mdm.Device
	case "User":
		return mdm.User
	case "User Enrollment (Device)":
		return mdm.UserEnrollmentDevice
	case "User Enrollment":
		return mdm.UserEnrollment
	case "Shared iPad":
		return mdm.SharediPad
	default:
		return 0
	}
}
