// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parserprovider

import (
	"context"
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGateFlagValue(t *testing.T) {
	flags := new(flag.FlagSet)
	Flags(flags)
	err := flags.Parse([]string{
		"--feature-gates=foo,-bar,+baz,foo",
	})
	require.NoError(t, err)

	pp := NewGatesMapProvider()
	cp, err := pp.Get(context.Background())
	require.NoError(t, err)
	keys := cp.AllKeys()
	assert.Len(t, keys, 3)
	assert.Equal(t, true, cp.Get("service::gates::foo"))
	assert.Equal(t, false, cp.Get("service::gates::bar"))
	assert.Equal(t, true, cp.Get("service::gates::baz"))
	require.NoError(t, pp.Close(context.Background()))

	assert.Equal(t, "-bar,baz,foo", gateFlag.String())
}
