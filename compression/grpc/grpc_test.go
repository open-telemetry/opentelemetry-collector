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

package grpc

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector/compression"
)

func TestGetGRPCCompressionKey(t *testing.T) {
	if GetGRPCCompressionKey("gzip") != compression.Gzip {
		t.Error("gzip is marked as supported but returned unsupported")
	}

	if GetGRPCCompressionKey("Gzip") != compression.Gzip {
		t.Error("Capitalization of Gzip should not matter")
	}

	if GetGRPCCompressionKey("badType") != compression.Unsupported {
		t.Error("badType is not supported but was returned as supported")
	}
}
