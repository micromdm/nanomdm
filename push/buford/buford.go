// Pacakge buford adapts the buford APNs push package to the PushProvider and
// PushProviderFactory interfaces.
package buford

import (
	"crypto/tls"
	"errors"
	"time"

	bufordpush "github.com/RobotsAndPencils/buford/push"
	"github.com/jessepeterson/nanomdm/mdm"
	"github.com/jessepeterson/nanomdm/push"
)

// bufordFactory instantiates new buford Services to satisfy the PushProviderFactory interface.
type bufordFactory struct {
	workers    uint
	expiration time.Time
}

// NewPushProviderFactory creates a new instance that can spawn buford Services
func NewPushProviderFactory() *bufordFactory {
	return &bufordFactory{
		workers: 5,
	}
}

// NewPushProvider generates a new PushProvider given a tls keypair
func (f *bufordFactory) NewPushProvider(cert *tls.Certificate) (push.PushProvider, error) {
	client, err := bufordpush.NewClient(*cert)
	if err != nil {
		return nil, err
	}
	prov := &bufordPushProvider{
		service: bufordpush.NewService(client, bufordpush.Production),
		workers: f.workers,
	}
	if !f.expiration.IsZero() {
		prov.headers = &bufordpush.Headers{Expiration: f.expiration}
	}
	return prov, err
}

// bufordPushProvider wraps a buford Service to satisfy the PushProvider interface.
type bufordPushProvider struct {
	service *bufordpush.Service
	headers *bufordpush.Headers
	workers uint
}

func (c *bufordPushProvider) pushSingle(pushInfo *mdm.Push) *push.Response {
	resp := new(push.Response)
	payload := []byte(`{"mdm":"` + pushInfo.PushMagic + `"}`)
	resp.Id, resp.Err = c.service.Push(pushInfo.Token.String(), c.headers, payload)
	return resp
}

func (c *bufordPushProvider) pushMulti(pushInfos []*mdm.Push) map[string]*push.Response {
	workers := uint(len(pushInfos))
	if workers > c.workers {
		workers = c.workers
	}
	queue := bufordpush.NewQueue(c.service, workers)
	defer queue.Close()
	for _, push := range pushInfos {
		payload := []byte(`{"mdm":"` + push.PushMagic + `"}`)
		go queue.Push(push.Token.String(), c.headers, payload)
	}
	responses := make(map[string]*push.Response)
	for range pushInfos {
		bufordResp := <-queue.Responses
		responses[bufordResp.DeviceToken] = &push.Response{
			Id:  bufordResp.ID,
			Err: bufordResp.Err,
		}
	}
	return responses
}

// Push sends 'raw' MDM APNs push notifications to service in c.
func (c *bufordPushProvider) Push(pushInfos []*mdm.Push) (map[string]*push.Response, error) {
	if len(pushInfos) < 1 {
		return nil, errors.New("no push data provided")
	}
	// some environments may heavily utilize individual pushes.
	// this justifies the special case and optimizes for it.
	if len(pushInfos) == 1 {
		responses := make(map[string]*push.Response)
		responses[pushInfos[0].Token.String()] = c.pushSingle(pushInfos[0])
		return responses, nil
	}
	return c.pushMulti(pushInfos), nil
}
