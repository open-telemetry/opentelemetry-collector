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
package fileexporter

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/internal"
	collectorlogs "go.opentelemetry.io/collector/internal/data/protogen/collector/logs/v1"
	collectormetrics "go.opentelemetry.io/collector/internal/data/protogen/collector/metrics/v1"
	collectortrace "go.opentelemetry.io/collector/internal/data/protogen/collector/trace/v1"
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestFileTracesExporter(t *testing.T) {
	fe := &fileExporter{path: tempFileName(t)}
	require.NotNil(t, fe)

	td := testdata.GenerateTracesTwoSpansSameResource()
	assert.NoError(t, fe.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, fe.ConsumeTraces(context.Background(), td))
	assert.NoError(t, fe.Shutdown(context.Background()))

	var unmarshaler = &jsonpb.Unmarshaler{}
	got := &collectortrace.ExportTraceServiceRequest{}
	buf, err := ioutil.ReadFile(fe.path)
	assert.NoError(t, err)
	assert.NoError(t, unmarshaler.Unmarshal(bytes.NewReader(buf), got))
	assert.EqualValues(t, internal.TracesToOtlp(td.InternalRep()), got)
}

func TestFileTracesExporterError(t *testing.T) {
	mf := &errorWriter{}
	fe := &fileExporter{file: mf}
	require.NotNil(t, fe)

	td := testdata.GenerateTracesTwoSpansSameResource()
	// Cannot call Start since we inject directly the WriterCloser.
	assert.Error(t, fe.ConsumeTraces(context.Background(), td))
	assert.NoError(t, fe.Shutdown(context.Background()))
}

func TestFileMetricsExporter(t *testing.T) {
	fe := &fileExporter{path: tempFileName(t)}
	require.NotNil(t, fe)

	md := testdata.GenerateMetricsTwoMetrics()
	assert.NoError(t, fe.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, fe.ConsumeMetrics(context.Background(), md))
	assert.NoError(t, fe.Shutdown(context.Background()))

	var unmarshaler = &jsonpb.Unmarshaler{}
	got := &collectormetrics.ExportMetricsServiceRequest{}
	buf, err := ioutil.ReadFile(fe.path)
	assert.NoError(t, err)
	assert.NoError(t, unmarshaler.Unmarshal(bytes.NewReader(buf), got))
	assert.EqualValues(t, internal.MetricsToOtlp(md.InternalRep()), got)
}

func TestFileMetricsExporterError(t *testing.T) {
	mf := &errorWriter{}
	fe := &fileExporter{file: mf}
	require.NotNil(t, fe)

	md := testdata.GenerateMetricsTwoMetrics()
	// Cannot call Start since we inject directly the WriterCloser.
	assert.Error(t, fe.ConsumeMetrics(context.Background(), md))
	assert.NoError(t, fe.Shutdown(context.Background()))
}

func TestFileLogsExporter(t *testing.T) {
	fe := &fileExporter{path: tempFileName(t)}
	require.NotNil(t, fe)

	otlp := testdata.GenerateLogsTwoLogRecordsSameResource()
	assert.NoError(t, fe.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, fe.ConsumeLogs(context.Background(), otlp))
	assert.NoError(t, fe.Shutdown(context.Background()))

	var unmarshaler = &jsonpb.Unmarshaler{}
	got := &collectorlogs.ExportLogsServiceRequest{}
	buf, err := ioutil.ReadFile(fe.path)
	assert.NoError(t, err)
	assert.NoError(t, unmarshaler.Unmarshal(bytes.NewReader(buf), got))
	assert.EqualValues(t, internal.LogsToOtlp(otlp.InternalRep()), got)
}

func TestFileLogsExporterErrors(t *testing.T) {
	mf := &errorWriter{}
	fe := &fileExporter{file: mf}
	require.NotNil(t, fe)

	otlp := testdata.GenerateLogsTwoLogRecordsSameResource()
	// Cannot call Start since we inject directly the WriterCloser.
	assert.Error(t, fe.ConsumeLogs(context.Background(), otlp))
	assert.NoError(t, fe.Shutdown(context.Background()))
}

// tempFileName provides a temporary file name for testing.
func tempFileName(t *testing.T) string {
	tmpfile, err := ioutil.TempFile("", "*.json")
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())
	socket := tmpfile.Name()
	require.NoError(t, os.Remove(socket))
	return socket
}

// errorWriter is an io.Writer that will return an error all ways
type errorWriter struct {
}

func (e errorWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("all ways return error")
}

func (e *errorWriter) Close() error {
	return nil
}
