// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"strconv"

	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/collector/featuregate"
)

// persistRequestContextFeatureGate controls whether request context should be persisted in the queue.
var persistRequestContextFeatureGate = featuregate.GlobalRegistry().MustRegister(
	"exporter.PersistRequestContext",
	featuregate.StageAlpha,
	featuregate.WithRegisterFromVersion("v0.127.0"),
	featuregate.WithRegisterDescription("controls whether context should be stored alongside requests in the persistent queue"),
	featuregate.WithRegisterReferenceURL("https://github.com/open-telemetry/opentelemetry-collector/pull/12934"),
)

var tracePropagator = propagation.TraceContext{}

func marshalSpanContext(ctx context.Context) []byte {
	if !persistRequestContextFeatureGate.IsEnabled() {
		return nil
	}
	carrier := newByteMapCarrier()
	tracePropagator.Inject(ctx, carrier)
	return carrier.Bytes()
}

func unmarshalSpanContext(b []byte) context.Context {
	ctx := context.Background()
	if !persistRequestContextFeatureGate.IsEnabled() || b == nil {
		return ctx
	}
	carrier := &byteMapCarrier{buf: b}
	tracePropagator.Extract(ctx, carrier)
	return ctx
}

func getContextKey(index uint64) string {
	return strconv.FormatUint(index, 10) + "_context"
}

// byteMapCarrier implements propagation.TextMapCarrier on top of a byte slice.
// The format is a sequence of key-value pairs encoded as:
//   - 1 byte length of key
//   - key string
//   - '=' character
//   - value string
//   - NUL terminator (0 byte)
type byteMapCarrier struct{ buf []byte }

var _ propagation.TextMapCarrier = (*byteMapCarrier)(nil)

// defaultCarrierCap is a capacity for the byteMapCarrier buffer that should fit typical `traceparent` and `tracestate`
// keys and values.
const defaultCarrierCap = 128

func newByteMapCarrier() *byteMapCarrier {
	return &byteMapCarrier{buf: make([]byte, 0, defaultCarrierCap)}
}

func (c *byteMapCarrier) Set(k, v string) {
	c.buf = append(c.buf, byte(len(k)))
	c.buf = append(c.buf, k...)
	c.buf = append(c.buf, '=')
	c.buf = append(c.buf, v...)
	c.buf = append(c.buf, 0) // NUL terminator
}

func (c *byteMapCarrier) Get(k string) string {
	for i := 0; i < len(c.buf); {
		l := int(c.buf[i])
		i++
		if i+l > len(c.buf) {
			return ""
		}
		key := string(c.buf[i : i+l])
		i += l
		if i >= len(c.buf) || c.buf[i] != '=' {
			return ""
		}
		i++
		valStart := i
		for i < len(c.buf) && c.buf[i] != 0 {
			i++
		}
		val := string(c.buf[valStart:i])
		i++ // skip NUL
		if key == k {
			return val
		}
	}
	return ""
}

func (c *byteMapCarrier) Keys() []string {
	var out []string
	for i := 0; i < len(c.buf); {
		l := int(c.buf[i])
		i++
		out = append(out, string(c.buf[i:i+l]))
		i += l
		for i < len(c.buf) && c.buf[i] != 0 {
			i++
		}
		i++ // skip '=' / NUL block
	}
	return out
}

func (c *byteMapCarrier) Bytes() []byte { return c.buf }
