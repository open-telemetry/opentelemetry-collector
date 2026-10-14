package confighttp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
)

func TestClientConfigUnmarshalRejectsUnknownKeys(t *testing.T) {
	cfg := NewDefaultClientConfig()
	conf := confmap.NewFromStringMap(map[string]any{
		"endpoint":   "http://localhost:4318",
		"compresion": "gzip",
	})
	err := conf.Unmarshal(&cfg)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid keys") || strings.Contains(err.Error(), "compresion"),
		"should report typo, got: %v", err)
}

func TestServerConfigUnmarshalRejectsUnknownKeys(t *testing.T) {
	cfg := NewDefaultServerConfig()
	conf := confmap.NewFromStringMap(map[string]any{
		"endpoint":      "0.0.0.0:4318",
		"tls_settings":  "plaintext",
	})
	err := conf.Unmarshal(&cfg)
	require.Error(t, err)
}
