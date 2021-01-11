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

package goldendataset

import (
	"testing"

	"github.com/stretchr/testify/assert"

	otlpresource "go.opentelemetry.io/collector/internal/data/protogen/resource/v1"
)

func TestGenerateResource(t *testing.T) {
	resourceIds := []PICTInputResource{ResourceNil, ResourceEmpty, ResourceVMOnPrem, ResourceVMCloud, ResourceK8sOnPrem,
		ResourceK8sCloud, ResourceFaas, ResourceExec}
	for _, rscID := range resourceIds {
		rsc := GenerateResource(rscID)
		if rscID == ResourceNil {
			assert.Nil(t, rsc.Attributes)
		} else {
			assert.NotNil(t, rsc.Attributes)
		}
		// test marshal/unmarshal
		bytes, err := rsc.Marshal()
		if err != nil {
			assert.Fail(t, err.Error())
		}
		if len(bytes) > 0 {
			copy := &otlpresource.Resource{}
			err = copy.Unmarshal(bytes)
			if err != nil {
				assert.Fail(t, err.Error())
			}
			assert.EqualValues(t, len(rsc.Attributes), len(copy.Attributes))
		}
	}
}
