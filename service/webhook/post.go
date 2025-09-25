package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func postWebhookEvent(
	ctx context.Context,
	client Doer,
	url string,
	event *EventJson,
) error {
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected HTTP status %d %s", resp.StatusCode, resp.Status)
	}
	return nil
}
