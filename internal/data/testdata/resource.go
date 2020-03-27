// Copyright 2020 OpenTelemetry Authors
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

package testdata

import (
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	otlpresource "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
)

func fillResource1(r data.Resource) {
	r.SetAttributes(generateResourceAttributes1())
}

func generateOtlpResource1() *otlpresource.Resource {
	return &otlpresource.Resource{
		Attributes: generateOtlpResourceAttributes1(),
	}
}

func fillResource2(r data.Resource) {
	r.SetAttributes(generateResourceAttributes2())
}

func generateOtlpResource2() *otlpresource.Resource {
	return &otlpresource.Resource{
		Attributes: generateOtlpResourceAttributes2(),
	}
}
