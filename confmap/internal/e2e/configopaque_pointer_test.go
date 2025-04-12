// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2etest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/configopaque"
)

func TestIssue12793_ConfigOpaquePointer(t *testing.T) {
    // Set up a test environment variable with a numeric value
    t.Setenv("VALUE", "12345")

    // Create a resolver with our test file
    resolver := NewResolver(t, "configopaque_pointer.yaml")
    conf, err := resolver.Resolve(context.Background())
    require.NoError(t, err)
    
    // Print the value from conf for debugging
    val := conf.Get("field")
    t.Logf("Value type: %T, value: %v", val, val)

    // Define our config struct with a *configopaque.String field
    type Config struct {
        Field *configopaque.String `mapstructure:"field"`
    }

    // Unmarshal into the config struct
    var cfg Config
    err = conf.Unmarshal(&cfg)
    
    // The test should pass after our fix
    require.NoError(t, err)
    require.NotNil(t, cfg.Field)
    assert.Equal(t, "12345", string(*cfg.Field))
}
