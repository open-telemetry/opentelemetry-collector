package summaryapireceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-service/internal/factories"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := factories.GetReceiverFactory(typeStr)
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
}

func TestCreateTraceReceiver(t *testing.T) {
	factory := factories.GetReceiverFactory(typeStr)
	cfg := factory.CreateDefaultConfig()

	tReceiver, err := factory.CreateTraceReceiver(context.Background(), cfg, nil)
	assert.Equal(t, err, factories.ErrDataTypeIsNotSupported)
	assert.Nil(t, tReceiver)
}
