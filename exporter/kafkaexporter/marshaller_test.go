// Copyright 2020 The OpenTelemetry Authors
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

package kafkaexporter

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestMarshall(t *testing.T) {
	pd := pdata.NewTraces()
	pd.ResourceSpans().Resize(1)
	pd.ResourceSpans().At(0).Resource().InitEmpty()
	pd.ResourceSpans().At(0).Resource().Attributes().InsertString("foo", "bar")
	m := protoMarshaller{}
	bts, err := m.Marshal(pd.ResourceSpans().At(0))
	require.NoError(t, err)
	expected, err := proto.Marshal(pdata.TracesToOtlp(pd)[0])
	require.NoError(t, err)
	assert.Equal(t, expected, bts)
}

func TestMarshall_empty(t *testing.T) {
	m := protoMarshaller{}
	bts, err := m.Marshal(pdata.NewResourceSpans())
	assert.Error(t, err)
	assert.Nil(t, bts)
}
