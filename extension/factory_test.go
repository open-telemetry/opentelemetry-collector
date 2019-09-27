// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package extension

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
)

type TestFactory struct {
	name string
}

// Type gets the type of the extension config created by this factory.
func (f *TestFactory) Type() string {
	return f.name
}

// CreateDefaultConfig creates the default configuration for the extension.
func (f *TestFactory) CreateDefaultConfig() configmodels.Extension {
	return nil
}

// CreateExtension creates a service extension based on this config.
func (f *TestFactory) CreateExtension(
	logger *zap.Logger,
	cfg configmodels.Extension,
) (ServiceExtension, error) {
	return nil, errors.New("cannot create extension from TestFactory")
}

func TestFactoriesBuilder(t *testing.T) {
	type testCase struct {
		in  []Factory
		out map[string]Factory
		err bool
	}

	testCases := []testCase{
		{
			in: []Factory{
				&TestFactory{"ext1"},
				&TestFactory{"ext2"},
			},
			out: map[string]Factory{
				"ext1": &TestFactory{"ext1"},
				"ext2": &TestFactory{"ext2"},
			},
			err: false,
		},
		{
			in: []Factory{
				&TestFactory{"ext1"},
				&TestFactory{"ext1"},
			},
			err: true,
		},
	}

	for _, c := range testCases {
		out, err := Build(c.in...)
		if c.err {
			assert.NotNil(t, err)
			continue
		}
		assert.Nil(t, err)
		assert.Equal(t, c.out, out)
	}
}
