package kv

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/micromdm/nanolib/storage/kv"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
)

const defaultEnrollmentsLimit = 100

// QueryEnrollments queries enrollments based on the provided request.
// Note: KV storage does not support efficient querying, so this implementation
// iterates through all enrollments and filters in memory.
func (s *KV) QueryEnrollments(ctx context.Context, req *storage.EnrollmentsQuery) (*storage.EnrollmentsQueryResult, error) {
	result := &storage.EnrollmentsQueryResult{
		Enrollments: make([]*storage.Enrollment, 0),
	}

	// Get pagination parameters
	offset, limit := 0, defaultEnrollmentsLimit
	if req != nil && req.Pagination != nil {
		offset, limit = req.Pagination.DefaultOffsetLimit(defaultEnrollmentsLimit)
	}

	// Collect all enrollment IDs by scanning for type keys
	enrollmentIDs, err := s.collectEnrollmentIDs(ctx)
	if err != nil {
		return nil, err
	}

	// Filter and paginate
	matchCount := 0
	for _, id := range enrollmentIDs {
		enrollment, err := s.loadEnrollment(ctx, id, req)
		if err != nil {
			continue // Skip enrollments we can't load
		}

		// Apply filters
		if !s.matchesFilter(enrollment, req) {
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

// collectEnrollmentIDs returns all enrollment IDs from the enrollments bucket.
func (s *KV) collectEnrollmentIDs(ctx context.Context) ([]string, error) {
	idSet := make(map[string]struct{})

	// Iterate through all keys in the enrollments bucket to find unique IDs
	// We look for keys that have the type suffix since every enrollment has a type
	for key := range s.enrollments.KeysPrefix(ctx, "", nil) {
		// Keys are in format "id.key_name"
		// Find the ID portion (everything before the last separator)
		if idx := strings.LastIndex(key, keySep); idx > 0 {
			id := key[:idx]
			// Check if this looks like an enrollment ID (not a nested key)
			if !strings.Contains(id, keySep) || strings.Count(id, keySep) == 0 {
				idSet[id] = struct{}{}
			} else {
				// For nested keys like "id.user_ch.subid", get the parent id
				parts := strings.Split(key, keySep)
				if len(parts) >= 2 {
					idSet[parts[0]] = struct{}{}
				}
			}
		}
	}

	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	return ids, nil
}

// loadEnrollment loads an enrollment by ID.
func (s *KV) loadEnrollment(ctx context.Context, id string, req *storage.EnrollmentsQuery) (*storage.Enrollment, error) {
	enrollment := &storage.Enrollment{
		ID: id,
	}

	// Load type
	typeBytes, err := s.enrollments.Get(ctx, join(id, keyEnrollmentType))
	if err != nil {
		if errors.Is(err, kv.ErrKeyNotFound) {
			return nil, errors.New("enrollment not found")
		}
		return nil, err
	}
	enrollment.Type = parseEnrollType(string(typeBytes))

	// Load enabled status (absence of disabled key means enabled)
	_, err = s.enrollments.Get(ctx, join(id, keyEnrollmentDisabled))
	enrollment.Enabled = errors.Is(err, kv.ErrKeyNotFound)

	// Load token update tally
	enrollment.TokenUpdateTally, _ = getTally(ctx, s.enrollments, id)

	// Load last seen
	lastSeenBytes, err := s.enrollments.Get(ctx, join(id, keyLastSeenAt))
	if err == nil {
		if ts, err := strconv.ParseInt(string(lastSeenBytes), 10, 64); err == nil {
			enrollment.LastSeen = time.UnixMicro(ts)
		}
	}

	// Load device data
	enrollment.Device = &storage.DeviceEnrollment{}

	// Get device ID (for user enrollments, use parent; for device enrollments, use self)
	deviceID := id
	if parentID, err := s.users.Get(ctx, join(id, keyUserDeviceChannel)); err == nil {
		deviceID = string(parentID)
		// This is a user enrollment, load user data
		enrollment.User = &storage.UserEnrollment{}
		// Note: user short/long names are not stored in KV backend
	}

	// Load serial number from devices bucket
	if serialBytes, err := s.devices.Get(ctx, join(deviceID, keyDeviceSerial)); err == nil {
		enrollment.Device.SerialNumber = string(serialBytes)
	}

	// Load optional data
	includeDeviceCert := false
	includeUnlockToken := false
	if req != nil && req.Options != nil {
		includeDeviceCert = req.Options.IncludeDeviceCert
		includeUnlockToken = req.Options.IncludeUnlockToken
	}

	if includeDeviceCert {
		if certBytes, err := s.devices.Get(ctx, join(deviceID, keyDeviceCert)); err == nil {
			// Convert DER to PEM format
			enrollment.Device.DeviceCert = string(certBytes) // Note: stored as DER in KV
		}
	}

	if includeUnlockToken {
		if unlockBytes, err := s.enrollments.Get(ctx, join(id, keyEnrollmentUnlockToken)); err == nil {
			enrollment.Device.UnlockToken = unlockBytes
		}
	}

	return enrollment, nil
}

// matchesFilter checks if an enrollment matches the query filter.
func (s *KV) matchesFilter(enrollment *storage.Enrollment, req *storage.EnrollmentsQuery) bool {
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
