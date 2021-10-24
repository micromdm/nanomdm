package nanomdm

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/service"
)

const enrollmentIDHeader = "X-Enrollment-ID"

type DeclarativeManagementHTTPCaller struct {
	urlPrefix string
	client    *http.Client
}

// NewDeclarativeManagementHTTPCaller creates a new DeclarativeManagementHTTPCaller
func NewDeclarativeManagementHTTPCaller(urlPrefix string) *DeclarativeManagementHTTPCaller {
	return &DeclarativeManagementHTTPCaller{
		urlPrefix: urlPrefix,
		client:    http.DefaultClient,
	}
}

// DeclarativeManagement calls out to an HTTP URL to handle the actual Declarative Management protocol
func (c *DeclarativeManagementHTTPCaller) DeclarativeManagement(r *mdm.Request, message *mdm.DeclarativeManagement) ([]byte, error) {
	if c.urlPrefix == "" {
		return nil, errors.New("missing URL")
	}
	u, err := url.Parse(c.urlPrefix)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, message.Endpoint)
	method := http.MethodGet
	if len(message.Data) > 0 {
		method = http.MethodPut
	}
	req, err := http.NewRequestWithContext(r.Context, method, u.String(), bytes.NewBuffer(message.Data))
	if err != nil {
		return nil, err
	}
	req.Header.Set(enrollmentIDHeader, r.ID)
	if len(message.Data) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return bodyBytes, service.NewHTTPStatusError(
			resp.StatusCode,
			fmt.Errorf("unexpected HTTP status: %s", resp.Status),
		)
	}
	return bodyBytes, nil
}
