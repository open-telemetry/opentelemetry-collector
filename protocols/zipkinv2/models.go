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
	"go.opentelemetry.io/collector/protocols/models"
	"go.opentelemetry.io/collector/translator/trace/zipkin"
)

var (
	_ models.TracesModelTranslator = (*Model)(nil)
)

type Model struct {
	// ParseStringTags is true if string tags should be automatically converted to numbers.
	ParseStringTags bool
}

func (z *Model) Type() interface{} {
	return []*zipkinmodel.SpanModel{}
}

func (z *Model) TracesFromModel(src interface{}) (pdata.Traces, error) {
	if model, ok := src.([]*zipkinmodel.SpanModel); ok {
		return zipkin.V2SpansToInternalTraces(model, z.ParseStringTags)
	}
	return pdata.NewTraces(), &models.ErrIncompatibleType{Model: src}
}

func (z *Model) TracesToModel(td pdata.Traces, out interface{}) error {
	if model, ok := out.(*[]*zipkinmodel.SpanModel); ok {
		sm, err := zipkin.InternalTracesToZipkinSpans(td)
		if err != nil {
			return err
		}
		*model = sm
	}
	return &models.ErrIncompatibleType{Model: out}
}
