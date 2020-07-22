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
	"fmt"

	"go.opentelemetry.io/collector/consumer/pdata"
)

// marshaller encodes traces into a byte array to be sent to Kafka.
type marshaller interface {
	// Marshal serializes spans into byte array
	Marshal(spans pdata.ResourceSpans) ([]byte, error)
}

type protoMarshaller struct {
}

func (m *protoMarshaller) Marshal(spans pdata.ResourceSpans) ([]byte, error) {
	pd := pdata.NewTraces()
	pd.ResourceSpans().Append(&spans)
	otlp := pdata.TracesToOtlp(pd)
	if len(otlp) < 1 || otlp[0].GetInstrumentationLibrarySpans() == nil && otlp[0].GetResource() == nil {
		return nil, fmt.Errorf("empty resource spans cannot be marshaled")
	}
	bts, err := otlp[0].Marshal()
	if err != nil {
		return nil, err
	}
	return bts, nil
}
