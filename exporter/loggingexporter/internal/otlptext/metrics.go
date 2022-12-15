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

package otlptext // import "go.opentelemetry.io/collector/exporter/loggingexporter/internal/otlptext"

import (
	"fmt"

	expohisto "github.com/lightstep/go-expohisto/mapping"
	"github.com/lightstep/go-expohisto/mapping/exponent"
	"github.com/lightstep/go-expohisto/mapping/logarithm"

	"go.opentelemetry.io/collector/pdata/pmetric"
)

// greatestBoundary equals and is the smallest unrepresentable float64
// greater than 1.  See TestLastBoundary for the definition in code.
const greatestBoundary = "1.79769e+308"

// boundaryFormat is used with Sprintf() to format exponential
// histogram boundaries.
const boundaryFormat = "%.6g"

// NewTextMetricsMarshaler returns a pmetric.Marshaler to encode to OTLP text bytes.
func NewTextMetricsMarshaler() pmetric.Marshaler {
	return textMetricsMarshaler{}
}

type textMetricsMarshaler struct{}

// MarshalMetrics pmetric.Metrics to OTLP text.
func (textMetricsMarshaler) MarshalMetrics(md pmetric.Metrics) ([]byte, error) {
	buf := dataBuffer{}
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		buf.logEntry("ResourceMetrics #%d", i)
		rm := rms.At(i)
		buf.logEntry("Resource SchemaURL: %s", rm.SchemaUrl())
		buf.logAttributes("Resource attributes", rm.Resource().Attributes())
		ilms := rm.ScopeMetrics()
		for j := 0; j < ilms.Len(); j++ {
			buf.logEntry("ScopeMetrics #%d", j)
			ilm := ilms.At(j)
			buf.logEntry("ScopeMetrics SchemaURL: %s", ilm.SchemaUrl())
			buf.logInstrumentationScope(ilm.Scope())
			metrics := ilm.Metrics()
			for k := 0; k < metrics.Len(); k++ {
				buf.logEntry("Metric #%d", k)
				metric := metrics.At(k)
				buf.logMetricDescriptor(metric)
				buf.logMetricDataPoints(metric)
			}
		}
	}

	return buf.buf.Bytes(), nil
}

type expoHistoMapping struct {
	scale   int32
	mapping expohisto.Mapping
}

func newExpoHistoMapping(scale int32) expoHistoMapping {
	m := expoHistoMapping{
		scale: scale,
	}
	if scale >= exponent.MinScale && scale <= exponent.MaxScale {
		m.mapping, _ = exponent.NewMapping(scale)
	} else if scale >= logarithm.MinScale && scale <= logarithm.MaxScale {
		m.mapping, _ = logarithm.NewMapping(scale)
	}
	return m
}

func (ehm expoHistoMapping) stringLowerBoundary(idx int32, neg bool) string {
	// Use the go-expohisto mapping functions provided the scale and
	// index are in range.
	if ehm.mapping != nil {
		if bound, err := ehm.mapping.LowerBoundary(idx); err == nil {
			// If the boundary is a subnormal number, fmt.Sprintf()
			// won't help, use the generic fallback below.
			if bound >= 0x1p-1022 {
				if neg {
					bound = -bound
				}
				return fmt.Sprintf(boundaryFormat, bound)
			}
		}
	}

	var s string
	switch {
	case idx == 0:
		s = "1"
	case idx > 0:
		// Note: at scale 20, the value (1<<30) leads to exponent 1024
		// The following expression generalizes this for valid scales.
		if ehm.scale >= -10 && ehm.scale <= 20 && int64(idx)<<(20-ehm.scale) == 1<<30 {
			// Important special case equal to 0x1p1024 is
			// the upper boundary of the last valid bucket
			// at all scales.
			s = greatestBoundary
		} else {
			s = "OVERFLOW"
		}
	default:
		// Note: corner cases involving subnormal values may
		// be handled here.  These are considered out of range
		// by the go-expohisto mapping functions, which will return
		// an underflow error for buckets that are entirely
		// outside the normal range.  These measurements are not
		// necessarily invalid, but they are extra work to compute.
		//
		// There is one case that falls through to this branch
		// to describe the lower boundary of a bucket
		// containing a normal value.  Because the value at
		// the subnormal boundary 0x1p-1022 falls into a
		// bucket (X, 0x1p-1022], where the subnormal value X
		// depends on scale.  The fallthrough here means we
		// print that bucket as "(UNDERFLOW, 2.22507e-308]",
		// (note 0x1p-1022 == 2.22507e-308).  This is correct
		// with respect to the reference implementation, which
		// first rounds subnormal values up to 0x1p-1022.  The
		// result (for the reference implementation) is all
		// subnormal values will fall into the bucket that
		// prints "(UNDERFLOW, 2.22507e-308]".
		s = "UNDERFLOW"
	}
	if neg {
		s = "-" + s
	}
	return s
}
