// Pacakge dump is a NanoMDM service that dumps raw responses
package dump

import (
	"encoding/base64"
	"fmt"
	"io"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/service"
)

type DumpWriter interface {
	io.Writer
	io.StringWriter
}

// Dumper is a service middleware that dumps MDM requests and responses
// to a file handle.
type Dumper struct {
	next service.CheckinAndCommandService
	w    DumpWriter
	cmd  bool
	bst  bool
	usr  bool
	dm   bool
}

// New creates a new dumper service middleware.
func New(next service.CheckinAndCommandService, w DumpWriter) *Dumper {
	return &Dumper{
		next: next,
		w:    w,
		cmd:  true,
		bst:  true,
		usr:  true,
		dm:   true,
	}
}

func (svc *Dumper) Authenticate(r *mdm.Request, m *mdm.Authenticate) error {
	svc.w.Write(m.Raw)
	return svc.next.Authenticate(r, m)
}

func (svc *Dumper) TokenUpdate(r *mdm.Request, m *mdm.TokenUpdate) error {
	svc.w.Write(m.Raw)
	return svc.next.TokenUpdate(r, m)
}

func (svc *Dumper) CheckOut(r *mdm.Request, m *mdm.CheckOut) error {
	svc.w.Write(m.Raw)
	return svc.next.CheckOut(r, m)
}

func (svc *Dumper) UserAuthenticate(r *mdm.Request, m *mdm.UserAuthenticate) ([]byte, error) {
	svc.w.Write(m.Raw)
	respBytes, err := svc.next.UserAuthenticate(r, m)
	if svc.usr && respBytes != nil && len(respBytes) > 0 {
		svc.w.Write(respBytes)
	}
	return respBytes, err
}

func (svc *Dumper) SetBootstrapToken(r *mdm.Request, m *mdm.SetBootstrapToken) error {
	svc.w.Write(m.Raw)
	return svc.next.SetBootstrapToken(r, m)
}

func (svc *Dumper) GetBootstrapToken(r *mdm.Request, m *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	svc.w.Write(m.Raw)
	bsToken, err := svc.next.GetBootstrapToken(r, m)
	if svc.bst && bsToken != nil && len(bsToken.BootstrapToken) > 0 {
		svc.w.Write([]byte(fmt.Sprintf("Bootstrap token: %s\n", bsToken.BootstrapToken.String())))
	}
	return bsToken, err
}

func (svc *Dumper) GetToken(r *mdm.Request, m *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	svc.w.Write(m.Raw)
	token, err := svc.next.GetToken(r, m)
	if token != nil && len(token.TokenData) > 0 {
		b64 := base64.StdEncoding.EncodeToString(token.TokenData)
		svc.w.WriteString("GetToken TokenData: " + b64 + "\n")
	}
	return token, err
}

func (svc *Dumper) CommandAndReportResults(r *mdm.Request, results *mdm.CommandResults) (*mdm.Command, error) {
	svc.w.Write(results.Raw)
	cmd, err := svc.next.CommandAndReportResults(r, results)
	if svc.cmd && err != nil && cmd != nil && cmd.Raw != nil {
		svc.w.Write(cmd.Raw)
	}
	return cmd, err
}

func (svc *Dumper) DeclarativeManagement(r *mdm.Request, m *mdm.DeclarativeManagement) ([]byte, error) {
	svc.w.Write(m.Raw)
	if len(m.Data) > 0 {
		svc.w.Write(m.Data)
	}
	respBytes, err := svc.next.DeclarativeManagement(r, m)
	if svc.dm && err != nil {
		svc.w.Write(respBytes)
	}
	return respBytes, err
}
