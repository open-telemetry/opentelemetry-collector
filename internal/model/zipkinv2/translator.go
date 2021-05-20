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

package zipkinv2

import (
	zipkinmodel "github.com/openzipkin/zipkin-go/model"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/model/translator"
	"go.opentelemetry.io/collector/translator/trace/zipkin"
)

var (
	_ translator.TracesEncoder = (*Model)(nil)
	_ translator.TracesDecoder = (*Model)(nil)
)

type Model struct {
	// ParseStringTags is true if string tags should be automatically converted to numbers.
	ParseStringTags bool
}

func (z *Model) ToTraces(src interface{}) (pdata.Traces, error) {
	if model, ok := src.([]*zipkinmodel.SpanModel); ok {
		return zipkin.V2SpansToInternalTraces(model, z.ParseStringTags)
	}
	return pdata.NewTraces(), translator.NewErrIncompatibleType([]*zipkinmodel.SpanModel{}, src)
}

func (z *Model) FromTraces(td pdata.Traces) (interface{}, error) {
	return zipkin.InternalTracesToZipkinSpans(td)
}
