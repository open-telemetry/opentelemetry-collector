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

package internal

import (
	"reflect"
	"testing"

	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"go.opencensus.io/resource/resourcekeys"
)

func Test_createNodeResource(t *testing.T) {
	resourceLabels := map[string]string{
		resourcekeys.CloudKeyZone:      "my_location",
		resourcekeys.K8SKeyClusterName: "my_cluster_name",
		resourcekeys.HostKeyName:       "my_node_name",
	}

	expected := &resourcepb.Resource{
		Type: resourcekeys.HostType,
		Labels: map[string]string{
			resourcekeys.CloudKeyZone:      "my_location",
			resourcekeys.K8SKeyClusterName: "my_cluster_name",
			resourcekeys.HostKeyName:       "my_node_name",
		},
	}
	result := createNodeResource(resourceLabels)
	if !reflect.DeepEqual(expected, result) {
		t.Errorf("Error: expected: %v, actual %v", expected, result)

	}
}

func Test_createNodeResourceNil(t *testing.T) {
	result := createNodeResource(nil)
	if result != nil {
		t.Errorf("Error: expected nil, actual %v", result)

	}
}

func Test_createNodeResourceMissingField(t *testing.T) {
	result1 := createNodeResource(map[string]string{
		resourcekeys.K8SKeyClusterName: "my_cluster_name",
		resourcekeys.HostKeyName:       "my_node_name",
	})
	if result1 != nil {
		t.Errorf("Error: expected nil, actual %v", result1)

	}
	result2 := createNodeResource(map[string]string{
		resourcekeys.CloudKeyZone: "my_location",
		resourcekeys.HostKeyName:  "my_node_name",
	})
	if result2 != nil {
		t.Errorf("Error: expected nil, actual %v", result2)

	}
	result3 := createNodeResource(map[string]string{
		resourcekeys.CloudKeyZone:      "my_location",
		resourcekeys.K8SKeyClusterName: "my_cluster_name",
	})
	if result3 != nil {
		t.Errorf("Error: expected nil, actual %v", result3)

	}
}
