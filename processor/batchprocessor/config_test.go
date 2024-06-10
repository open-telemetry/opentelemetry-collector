// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/apple/pkl-go/pkl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestUnmarshalDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, confmap.New().Unmarshal(&cfg))
	assert.Equal(t, factory.CreateDefaultConfig(), cfg)
}

func TestUnmarshalConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	got := factory.CreateDefaultConfig()
	assert.NoError(t, cm.Unmarshal(&got))
	want := &Config{
		SendBatchSize:            uint32(10000),
		SendBatchMaxSize:         uint32(11000),
		Timeout:                  &pkl.Duration{Value: 10, Unit: pkl.Second},
		MetadataCardinalityLimit: 1000,
		MetadataKeys:             []string{},
	}
	assert.True(t, reflect.DeepEqual(got, want), "want: %+v got: %+v", want, got)
}

// func TestValidateConfig_DefaultBatchMaxSize(t *testing.T) {
// 	cfg := &Config{
// 		SendBatchSize:    100,
// 		SendBatchMaxSize: 0,
// 	}
// 	assert.NoError(t, cfg.Validate())
// }

// func TestValidateConfig_ValidBatchSizes(t *testing.T) {
// 	cfg := &Config{
// 		SendBatchSize:    100,
// 		SendBatchMaxSize: 1000,
// 	}
// 	assert.NoError(t, cfg.Validate())

// }

// func TestValidateConfig_InvalidBatchSize(t *testing.T) {
// 	cfg := &Config{
// 		SendBatchSize:    1000,
// 		SendBatchMaxSize: 100,
// 	}
// 	assert.Error(t, cfg.Validate())
// }

// func TestValidateConfig_InvalidTimeout(t *testing.T) {
// 	cfg := &Config{
// 		Timeout: &pkl.Duration{Value: -1, Unit: pkl.Second},
// 	}
// 	assert.Error(t, cfg.Validate())
// }

// func TestValidateConfig_ValidZero(t *testing.T) {
// 	cfg := &Config{}
// 	assert.NoError(t, cfg.Validate())
// }
