package summaryapireceiver

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-service/internal/configmodels"
	"github.com/open-telemetry/opentelemetry-service/internal/configv2"
	"github.com/open-telemetry/opentelemetry-service/internal/factories"
)

var _ = configv2.RegisterTestFactories()

func TestLoadConfig(t *testing.T) {
	factory := factories.GetReceiverFactory(typeStr)

	config, err := configv2.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"))

	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, len(config.Receivers), 2)

	r0 := config.Receivers["summaryapi"]
	assert.Equal(t, r0, factory.CreateDefaultConfig())

	r1 := config.Receivers["summaryapi/customname"].(*ConfigV2)
	assert.Equal(t, r1,
		&ConfigV2{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal:  typeStr,
				NameVal:  "summaryapi/customname",
				Endpoint: "127.0.0.1:9090",
			},
		})
}
