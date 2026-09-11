package file

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
)

const defaultEnrollmentsLimit = 100

// QueryEnrollments queries enrollments based on the provided request.
// Note: File storage does not support efficient querying, so this implementation
// iterates through all enrollment directories and filters in memory.
func (s *FileStorage) QueryEnrollments(ctx context.Context, req *storage.EnrollmentsQuery) (*storage.EnrollmentsQueryResult, error) {
	result := &storage.EnrollmentsQueryResult{
		Enrollments: make([]*storage.Enrollment, 0),
	}

	// Get pagination parameters
	offset, limit := 0, defaultEnrollmentsLimit
	if req != nil && req.Pagination != nil {
		offset, limit = req.Pagination.DefaultOffsetLimit(defaultEnrollmentsLimit)
	}

	// List all enrollment directories
	entries, err := os.ReadDir(s.path)
	if err != nil {
		return nil, err
	}

	// Filter and paginate
	matchCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		id := entry.Name()
		enrollment, err := s.loadEnrollment(ctx, id, req)
		if err != nil {
			continue // Skip enrollments we can't load
		}

		// Apply filters
		if !matchesFilter(enrollment, req) {
			continue
		}

		matchCount++

		// Apply offset
		if matchCount <= offset {
			continue
		}

		// Apply limit
		if len(result.Enrollments) >= limit {
			break
		}

		result.Enrollments = append(result.Enrollments, enrollment)
	}

	return result, nil
}

// loadEnrollment loads an enrollment by ID.
func (s *FileStorage) loadEnrollment(ctx context.Context, id string, req *storage.EnrollmentsQuery) (*storage.Enrollment, error) {
	e := s.newEnrollment(id)

	// Check if this is a valid enrollment by checking for TokenUpdate file
	exists, err := e.fileExists(TokenUpdateFilename)
	if err != nil {
		return nil, err
	}
	if !exists {
		// Also check for Authenticate file
		exists, err = e.fileExists(AuthenticateFilename)
		if err != nil || !exists {
			return nil, errors.New("not a valid enrollment")
		}
	}

	enrollment := &storage.Enrollment{
		ID: id,
	}

	// Determine type based on directory structure
	// User enrollments have a parent device association
	enrollment.Type = mdm.Device // Default to device

	// Check if disabled
	disabled, _ := e.fileExists(DisabledFilename)
	enrollment.Enabled = !disabled

	// Load token update tally
	enrollment.TokenUpdateTally, _ = e.readNumericFile(TokenUpdateTallyFilename)

	// Load last seen (use TokenUpdate file modification time as proxy)
	if info, err := os.Stat(e.dirPrefix(TokenUpdateFilename)); err == nil {
		enrollment.LastSeen = info.ModTime()
	}

	// Load device data
	enrollment.Device = &storage.DeviceEnrollment{}

	// Load serial number
	if serialBytes, err := e.readFile(SerialNumberFilename); err == nil {
		enrollment.Device.SerialNumber = strings.TrimSpace(string(serialBytes))
	}

	// Load optional data
	includeDeviceCert := false
	includeUnlockToken := false
	if req != nil && req.Options != nil {
		includeDeviceCert = req.Options.IncludeDeviceCert
		includeUnlockToken = req.Options.IncludeUnlockToken
	}

	if includeDeviceCert {
		if certBytes, err := e.readFile(IdentityCertFilename); err == nil {
			enrollment.Device.DeviceCert = string(certBytes)
		}
	}

	if includeUnlockToken {
		if unlockBytes, err := e.readFile(UnlockTokenFilename); err == nil {
			enrollment.Device.UnlockToken = unlockBytes
		}
	}

	// Check for sub-enrollments (user channels)
	subEnrollmentPath := filepath.Join(e.dir(), SubEnrollmentPathname)
	if subEntries, err := os.ReadDir(subEnrollmentPath); err == nil && len(subEntries) > 0 {
		// This device has user channel enrollments
		// The sub-enrollments themselves are loaded separately
	}

	return enrollment, nil
}

// matchesFilter checks if an enrollment matches the query filter.
func matchesFilter(enrollment *storage.Enrollment, req *storage.EnrollmentsQuery) bool {
	if req == nil || req.Filter == nil {
		return true
	}

	f := req.Filter

	// Filter by IDs
	if len(f.IDs) > 0 {
		found := false
		for _, id := range f.IDs {
			if enrollment.ID == id {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by serials
	if len(f.Serials) > 0 {
		if enrollment.Device == nil {
			return false
		}
		found := false
		for _, serial := range f.Serials {
			if enrollment.Device.SerialNumber == serial {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by user short names
	if len(f.UserShortNames) > 0 {
		if enrollment.User == nil {
			return false
		}
		found := false
		for _, name := range f.UserShortNames {
			if enrollment.User.UserShortName == name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by types
	if len(f.Types) > 0 {
		found := false
		for _, t := range f.Types {
			if enrollment.Type.String() == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by enabled status
	if f.Enabled != nil && enrollment.Enabled != *f.Enabled {
		return false
	}

	return true
}
