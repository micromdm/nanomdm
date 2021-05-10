// Pacakge nanomdm is an MDM service.
package nanomdm

import (
	"fmt"

	"github.com/jessepeterson/nanomdm/log"
	"github.com/jessepeterson/nanomdm/mdm"
	"github.com/jessepeterson/nanomdm/storage"
)

type Service struct {
	logger     log.Logger
	normalizer func(e *mdm.Enrollment) *mdm.EnrollID
	store      storage.ServiceStore
}

// normalize generates an EnrollID from an Enrollment. Importantly
// we define what both device- and user-channel unique identifiers look
// like. Note ParentID field needs to contain what the same identifier
// for a non-user-channel type be for ID.
func normalize(e *mdm.Enrollment) *mdm.EnrollID {
	r := e.Resolved()
	if r == nil {
		return nil
	}
	eid := &mdm.EnrollID{
		Type: r.Type,
		ID:   r.DeviceChannelID,
	}
	if r.IsUserChannel {
		eid.ID += ":" + r.UserChannelID
		eid.ParentID = r.DeviceChannelID
	}
	return eid
}

func New(store storage.ServiceStore, logger log.Logger) *Service {
	return &Service{
		store:      store,
		logger:     logger,
		normalizer: normalize,
	}
}

func (s *Service) updateEnrollmentID(r *mdm.Request, e *mdm.Enrollment) error {
	if r.EnrollID != nil && r.ID != "" {
		s.logger.Debug("msg", "overwriting enrollment id")
	}
	r.EnrollID = s.normalizer(e)
	return r.EnrollID.Validate()
}

func (s *Service) Authenticate(r *mdm.Request, message *mdm.Authenticate) error {
	if err := s.updateEnrollmentID(r, &message.Enrollment); err != nil {
		return err
	}
	logs := []interface{}{
		"msg", "Authenticate",
		"id", r.ID,
		"type", r.Type,
	}
	if message.SerialNumber != "" {
		logs = append(logs, "serial_number", message.SerialNumber)
	}
	s.logger.Info(logs...)
	if err := s.store.StoreAuthenticate(r, message); err != nil {
		return err
	}
	// clear the command queue for any enrollment or sub-enrollment
	// this prevents queued commands still being queued after device
	// unenrollment
	if err := s.store.ClearQueue(r); err != nil {
		return err
	}
	// then disable the enrollment or any sub-enrollment (because an
	// enrollment is only valid after a tokenupdate)
	return s.store.Disable(r)
}

func (s *Service) TokenUpdate(r *mdm.Request, message *mdm.TokenUpdate) error {
	if err := s.updateEnrollmentID(r, &message.Enrollment); err != nil {
		return err
	}
	s.logger.Info(
		"msg", "TokenUpdate",
		"id", r.ID,
		"type", r.Type,
	)
	return s.store.StoreTokenUpdate(r, message)
}

func (s *Service) CheckOut(r *mdm.Request, message *mdm.CheckOut) error {
	if err := s.updateEnrollmentID(r, &message.Enrollment); err != nil {
		return err
	}
	s.logger.Info(
		"msg", "CheckOut",
		"id", r.ID,
		"type", r.Type,
	)
	// disable an enrollment upon checkout
	return s.store.Disable(r)
}

func (s *Service) CommandAndReportResults(r *mdm.Request, results *mdm.CommandResults) (*mdm.Command, error) {
	if err := s.updateEnrollmentID(r, &results.Enrollment); err != nil {
		return nil, err
	}
	logs := []interface{}{
		"status", results.Status,
		"id", r.ID,
		"type", r.Type,
	}
	if results.Status != "Idle" {
		logs = append(logs, "command_uuid", results.CommandUUID)
	}
	s.logger.Info(logs...)
	err := s.store.StoreCommandReport(r, results)
	if err != nil {
		return nil, fmt.Errorf("storing command report: %w", err)
	}
	cmd, err := s.store.RetrieveNextCommand(r, results.Status == "NotNow")
	if err != nil {
		return nil, fmt.Errorf("retrieving next command: %w", err)
	}
	if cmd != nil {
		s.logger.Debug(
			"msg", "command retrieved",
			"id", r.ID,
			"command_uuid", cmd.CommandUUID,
		)
		return cmd, nil
	}
	s.logger.Debug(
		"msg", "no command retrieved",
		"id", r.ID,
	)
	return nil, nil
}
