// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSliceOfMapsUnmarshal(t *testing.T) {
	// Config struct with a slice of maps
	type Header struct {
		Action       string `mapstructure:"action"`
		Key          string `mapstructure:"key"`
		FromContext  string `mapstructure:"from_context"`
		DefaultValue string `mapstructure:"default_value"`
	}

	type HeadersSetterConfig struct {
		Headers []Header `mapstructure:"headers"`
	}

	// Create a confmap with a numeric value in a map within a slice
	conf := NewFromStringMap(map[string]interface{}{
		"headers": []interface{}{
			map[string]interface{}{
				"action":        "upsert",
				"key":           "X-TOKEN",
				"from_context":  "X-TOKEN",
				"default_value": 12345, // Numeric value that should be converted to string
			},
		},
	})

	// Unmarshal into the config struct
	var config HeadersSetterConfig
	err := conf.Unmarshal(&config)
	require.NoError(t, err)

	// Verify the numeric value was correctly converted to a string
	assert.Len(t, config.Headers, 1)
	assert.Equal(t, "12345", config.Headers[0].DefaultValue)
}
