package microwebhook

import (
	"encoding/json"

	"github.com/micromdm/nanomdm/factories"
)

func postWebhookEventAMQP(
	amqpClient *factories.QueueFactory,
	exchange string,
	routingKey string,
	event *Event,
) error {
	jsonBytes, err := json.MarshalIndent(event, "", "\t")
	if err != nil {
		return err
	}

	err = amqpClient.PublishMessage(factories.QueueMessage{
		RoutingKey: routingKey,
		Exchange:   exchange,
		Type:       "nanomdm.event",
		Data:       jsonBytes,
	})
	if err != nil {
		return err
	}
	return nil
}
