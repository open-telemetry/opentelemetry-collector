package consumerhelper

import (
	"go.opentelemetry.io/collector/consumer"
)

type baseConsumer struct {
	capabilities consumer.Capabilities
}

func (bc *baseConsumer) GetCapabilities() consumer.Capabilities {
	return bc.capabilities
}

// NewConsumer returns a consumer.Consumer that implements the basic consumer functionality.
func NewConsumer(capabilities consumer.Capabilities) consumer.Consumer {
	return &baseConsumer{
		capabilities: capabilities,
	}
}
