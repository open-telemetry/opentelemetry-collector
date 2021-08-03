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

package awss3configsource

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestS3SessionForKVRetrieval(t *testing.T) {
	logger := zap.NewNop()

	config := Config{
		Region: "us-west-2",
		Bucket: "hasconfig",
		Key:    "retest_config.yaml",
	}

	cs, err := newConfigSource(logger, &config)
	require.NoError(t, err)
	require.NotNil(t, cs)

	s, err := cs.NewSession(context.Background())
	require.NoError(t, err)
	require.NotNil(t, s)

	retreived, err := s.Retrieve(context.Background(), "exporters::logging::loglevel", nil)
	require.NoError(t, err)
	require.Equal(t, "debug", retreived.Value().(string))
}
