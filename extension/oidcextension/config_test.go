package oidcextension

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/configtest"
	"path"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Extensions[typeStr] = factory
	cfg, err := configtest.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)

	require.Nil(t, err)
	require.NotNil(t, cfg)

	ext0 := cfg.Extensions["oidc"]
	assert.Equal(t, factory.CreateDefaultConfig(), ext0)

	ext1 := cfg.Extensions["oidc/1"]
	assert.Equal(t,
		&Config{
			ExtensionSettings: configmodels.ExtensionSettings{
				TypeVal: "oidc",
				NameVal: "oidc/1",
			},
			Attribute: "authorization",
			IssuerCAPath: "/etc/pki/tls/cert.pem",
			IssuerURL: "https://auth.example.com/",
			UsernameClaim: "email",
			GroupsClaim: "group",
			Audience: "my-oidc-client",
		},
		ext1)

	assert.Equal(t, 1, len(cfg.Service.Extensions))
	assert.Equal(t, "oidc/1", cfg.Service.Extensions[0])
}
