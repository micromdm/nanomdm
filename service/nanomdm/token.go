package nanomdm

import (
	"fmt"
	"sync"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/service"
)

type StaticToken struct {
	token []byte
}

func NewStaticToken(token []byte) *StaticToken {
	return &StaticToken{token: token}
}

func (t *StaticToken) GetToken(_ *mdm.Request, _ *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	return &mdm.GetTokenResponse{TokenData: t.token}, nil
}

// TokenServiceTypeMux is a middleware multiplexer for GetToken check-in messages.
// A TokenServiceType string is associated with a GetToken handler and
// then dispatched appropriately with a matching TokenServiceType.
type TokenServiceTypeMux struct {
	typesMu sync.RWMutex
	types   map[string]service.GetToken
}

// NewTokenServiceTypeMux creates a new TokenServiceTypeMux.
func NewTokenServiceTypeMux() *TokenServiceTypeMux { return &TokenServiceTypeMux{} }

// Handle registers a GetToken handler for the given service type.
// See https://developer.apple.com/documentation/devicemanagement/gettokenrequest
func (mux *TokenServiceTypeMux) Handle(serviceType string, handler service.GetToken) {
	if serviceType == "" {
		panic("tokenmux: invalid service type")
	}
	if handler == nil {
		panic("tokenmux: invalid handler")
	}
	mux.typesMu.Lock()
	defer mux.typesMu.Unlock()
	if mux.types == nil {
		mux.types = make(map[string]service.GetToken)
	} else if _, exists := mux.types[serviceType]; exists {
		panic("tokenmux: multiple registrations for " + serviceType)
	}
	mux.types[serviceType] = handler
}

// GetToken is the middleware that dispatches a GetToken handler based on service type.
func (mux *TokenServiceTypeMux) GetToken(r *mdm.Request, t *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	var next service.GetToken
	var serviceType string
	if t != nil {
		serviceType = t.TokenServiceType
	}
	mux.typesMu.RLock()
	if mux.types != nil {
		next = mux.types[serviceType]
	}
	mux.typesMu.RUnlock()
	if next == nil {
		return nil, fmt.Errorf("no handler for TokenServiceType: %v", serviceType)
	}
	return next.GetToken(r, t)
}
