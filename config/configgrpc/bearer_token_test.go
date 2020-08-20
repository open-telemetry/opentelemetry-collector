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

package configgrpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBearerToken(t *testing.T) {
	// test
	result := BearerToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...")
	metadata, err := result.GetRequestMetadata(context.Background())
	require.NoError(t, err)

	// verify
	assert.Equal(t, "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...", metadata["authorization"])
}

func TestBearerTokenRequiresSecureTransport(t *testing.T) {
	// test
	token := BearerToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...")

	// verify
	assert.True(t, token.RequireTransportSecurity())
}
