// Pacakge dump is a NanoMDM service that dumps raw responses
package dump

import (
	"os"

	"github.com/jessepeterson/nanomdm/mdm"
	"github.com/jessepeterson/nanomdm/service"
)

// Dumper is a service middleware that dumps MDM requests and responses
// to a file handle.
type Dumper struct {
	next service.CheckinAndCommandService
	file *os.File
	cmd  bool
}

// New creates a new dumper service middleware.
func New(next service.CheckinAndCommandService, file *os.File) *Dumper {
	return &Dumper{
		next: next,
		file: file,
		cmd:  true,
	}
}

func (svc *Dumper) Authenticate(r *mdm.Request, m *mdm.Authenticate) error {
	svc.file.Write(m.Raw)
	return svc.next.Authenticate(r, m)
}

func (svc *Dumper) TokenUpdate(r *mdm.Request, m *mdm.TokenUpdate) error {
	svc.file.Write(m.Raw)
	return svc.next.TokenUpdate(r, m)
}

func (svc *Dumper) CommandAndReportResults(r *mdm.Request, results *mdm.CommandResults) (*mdm.Command, error) {
	svc.file.Write(results.Raw)
	cmd, err := svc.next.CommandAndReportResults(r, results)
	if svc.cmd && err != nil && cmd != nil && cmd.Raw != nil {
		svc.file.Write(cmd.Raw)
	}
	return cmd, err
}
