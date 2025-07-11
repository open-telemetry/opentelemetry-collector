package ptrace

import (
	"bytes"

	"go.opentelemetry.io/collector/pdata/internal"
	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
)

// JSONMarshaler marshals pdata.Logs to JSON bytes using the OTLP/JSON format.
type JSONMarshaler struct{}

// MarshalLogs to the OTLP/JSON format.
func (*JSONMarshaler) MarshalTraces(td Traces) ([]byte, error) {
	buf := bytes.Buffer{}
	pb := internal.TracesToProto(internal.Traces(td))
	err := json.Marshal(&buf, &pb)
	return buf.Bytes(), err
}

var _ Unmarshaler = (*JSONUnmarshaler)(nil)

type JSONUnmarshaler struct{}

func (d *JSONUnmarshaler) UnmarshalTraces(buf []byte) (Traces, error) {
	td := otlptrace.TracesData{}
	if err := json.Unmarshal(bytes.NewReader(buf), &td); err != nil {
		return Traces{}, err
	}

	otlp.MigrateTraces(td.ResourceSpans)
	return Traces(internal.TracesFromProto(td)), nil
}
