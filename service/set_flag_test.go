// Copyright  The OpenTelemetry Authors
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

package service

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config"
)

func TestSetFlags(t *testing.T) {
	cmd := &cobra.Command{}
	addSetFlag(cmd.Flags())

	err := cmd.ParseFlags([]string{
		"--set=processors.batch.timeout=2s",
		"--set=processors.batch/foo.timeout=3s",
		"--set=receivers.otlp.protocols.grpc.endpoint=localhost:1818",
		"--set=exporters.kafka.brokers=foo:9200,foo2:9200",
	})
	require.NoError(t, err)

	cp := config.NewParser()
	err = AddSetFlagProperties(cp, cmd)
	require.NoError(t, err)

	keys := cp.AllKeys()
	assert.Len(t, keys, 4)
	assert.Equal(t, "2s", cp.Get("processors::batch::timeout"))
	assert.Equal(t, "3s", cp.Get("processors::batch/foo::timeout"))
	assert.Equal(t, "foo:9200,foo2:9200", cp.Get("exporters::kafka::brokers"))
	assert.Equal(t, "localhost:1818", cp.Get("receivers::otlp::protocols::grpc::endpoint"))
}

func TestSetFlags_err_set_flag(t *testing.T) {
	cmd := &cobra.Command{}
	require.Error(t, AddSetFlagProperties(config.NewParser(), cmd))
}

func TestSetFlags_empty(t *testing.T) {
	cmd := &cobra.Command{}
	addSetFlag(cmd.Flags())
	cp := config.NewParser()
	err := AddSetFlagProperties(cp, cmd)
	require.NoError(t, err)
	assert.Equal(t, 0, len(cp.AllKeys()))
}
