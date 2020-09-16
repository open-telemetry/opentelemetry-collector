// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package groupbytraceprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
)

func TestDefaultConfiguration(t *testing.T) {
	// test
	c := createDefaultConfig().(*Config)

	// verify
	assert.Equal(t, defaultNumTraces, c.NumTraces)
	assert.Equal(t, defaultWaitDuration, c.WaitDuration)
	assert.Equal(t, defaultDiscardOrphans, c.DiscardOrphans)
	assert.Equal(t, defaultStoreOnDisk, c.StoreOnDisk)
}

func TestCreateTestProcessor(t *testing.T) {
	c := createDefaultConfig().(*Config)

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	params := component.ProcessorCreateParams{
		Logger: logger,
	}
	next := &mockProcessor{}

	// test
	p, err := createTraceProcessor(context.Background(), params, c, next)

	// verify
	assert.NoError(t, err)
	assert.NotNil(t, p)
}

func TestCreateTestProcessorWithNotImplementedOptions(t *testing.T) {
	// prepare
	f := NewFactory()
	params := component.ProcessorCreateParams{}
	next := &mockProcessor{}

	// test
	for _, tt := range []struct {
		config      *Config
		expectedErr error
	}{
		{
			&Config{
				DiscardOrphans: true,
			},
			errDiscardOrphansNotSupported,
		},
		{
			&Config{
				StoreOnDisk: true,
			},
			errDiskStorageNotSupported,
		},
	} {
		p, err := f.CreateTraceProcessor(context.Background(), params, tt.config, next)

		// verify
		assert.Error(t, tt.expectedErr, err)
		assert.Nil(t, p)
	}
}
