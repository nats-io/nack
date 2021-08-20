package jsm

import (
	"github.com/nats-io/jsm.go/api"
)

// ParseEvent parses event e and returns event as for example *api.ConsumerAckMetric, all unknown
// event schemas will be of type *UnknownMessage
func ParseEvent(e []byte) (schema string, event interface{}, err error) {
	return api.ParseMessage(e)
}
