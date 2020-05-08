// Copyright 2020, OpenTelemetry Authors
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

package golden_dataset

import (
	"testing"

	"github.com/golang/protobuf/proto"
	otlpresource "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
	"github.com/stretchr/testify/assert"
)

func TestGenerateResource(t *testing.T) {
	resourceIds := []string{ResourceNil, ResourceEmpty, ResourceVMOnPrem, ResourceVMCloud, ResourceK8sOnPrem,
		ResourceK8sCloud, ResourceFaas}
	for _, rscId := range resourceIds {
		rsc := GenerateResource(rscId)
		str := rsc.String()
		t.Logf("Resource: %s", str)
		if rscId == ResourceNil {
			assert.Nil(t, rsc.Attributes)
		} else {
			assert.NotNil(t, rsc.Attributes)
		}
		// test marshal/unmarshal
		bytes, err := proto.Marshal(rsc)
		if err != nil {
			assert.Fail(t, err.Error())
		}
		if len(bytes) > 0 {
			copy := &otlpresource.Resource{}
			err = proto.Unmarshal(bytes, copy)
			if err != nil {
				assert.Fail(t, err.Error())
			}
			assert.EqualValues(t, len(rsc.Attributes), len(copy.Attributes))
		}
	}
}
