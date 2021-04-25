// Copyright The OpenTelemetry Authors
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

package prometheusremotewriteexporter

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

func TestWALCreation_nilConfig(t *testing.T) {
	config := (*walConfig)(nil)
	wal, err := config.createWAL()
	assert.Nil(t, wal)
	assert.Nil(t, err)
}

func TestWALCreation_nonNilConfig(t *testing.T) {
	config := &walConfig{Dir: t.TempDir()}
	wal, err := config.createWAL()
	assert.NotNil(t, wal)
	assert.Nil(t, err)
	wal.Close()
}

var defaultBuildInfo = component.BuildInfo{
	Description: "OpenTelemetry Collector",
	Version:     "1.0",
}

// Ensures that when we attach the Write-Ahead-Log(WAL) to the exporter,
// that it successfully writes the serialized prompb.WriteRequests to the WAL,
// and that we can retrieve those exact requests back from the WAL, when the
// exporter starts up once again, that it picks up where it left off.
func TestWALOnExporterRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("This could be a long running test")
	}

	// 1. Create the WAL configuration, create the
	// exporter and export some time series!
	tempDir := t.TempDir()
	yamlConfig := fmt.Sprintf(`
wal:
    directory:           %s
    truncate_frequency:  60us
    cache_size:          1
        `, tempDir)

	parser, err := config.NewParserFromBuffer(strings.NewReader(yamlConfig))
	require.Nil(t, err)

	fullConfig := new(Config)
	if err = parser.UnmarshalExact(fullConfig); err != nil {
		t.Fatalf("Failed to parse config from YAML: %v", err)
	}
	walConfig := fullConfig.WALConfig
	require.NotNil(t, walConfig)
	require.Equal(t, walConfig.RefreshDuration, 60*time.Microsecond)
	require.Equal(t, walConfig.Dir, tempDir)

	// 1. Create a mock Prometheus Remote Write Exporter that'll just
	// receive the bytes uploaded to it by our exporter.
	uploadedBytesCh := make(chan []byte, 1)
	exiting := make(chan bool)
	prweServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		uploaded, err2 := ioutil.ReadAll(req.Body)
		assert.Nil(t, err2, "Error while reading from HTTP upload")
		select {
		case uploadedBytesCh <- uploaded:
		case <-exiting:
			return
		}
	}))
	defer prweServer.Close()

	prwe, err := NewPrwExporter("test_ns", prweServer.URL, prweServer.Client(), nil, 1, defaultBuildInfo, walConfig)
	assert.Nil(t, err)
	ctx := context.Background()
	defer prwe.Shutdown(ctx)
	defer close(exiting)

	assert.NotNil(t, prwe.wal)

	ts1 := &prompb.TimeSeries{
		Labels:  []prompb.Label{{Name: "ts1l1", Value: "ts1k1"}},
		Samples: []prompb.Sample{{Value: 1, Timestamp: 100}},
	}
	ts2 := &prompb.TimeSeries{
		Labels:  []prompb.Label{{Name: "ts2l1", Value: "ts2k1"}},
		Samples: []prompb.Sample{{Value: 2, Timestamp: 200}},
	}
	tsMap := map[string]*prompb.TimeSeries{
		"timeseries1": ts1,
		"timeseries2": ts2,
	}
	errs := prwe.export(ctx, tsMap)
	assert.Nil(t, errs)
	// Shutdown after we've written to the WAL.
	prwe.Shutdown(ctx)

	// 2. Let's now read back all of the WAL records and ensure
	// that all the prompb.WriteRequest values exist as we sent them.
	wal, err := walConfig.createWAL()
	assert.Nil(t, err)
	assert.NotNil(t, wal)
	defer wal.Close()

	// Read all the indices.
	firstIndex, err := wal.FirstIndex()
	assert.Nil(t, err)
	lastIndex, err := wal.LastIndex()
	assert.Nil(t, err)

	var reqs []*prompb.WriteRequest
	for i := firstIndex; i <= lastIndex; i++ {
		protoBlob, perr := wal.Read(i)
		assert.Nil(t, perr)
		assert.NotNil(t, protoBlob)
		req := new(prompb.WriteRequest)
		err = proto.Unmarshal(protoBlob, req)
		assert.Nil(t, err)
		reqs = append(reqs, req)
	}
	assert.Equal(t, 1, len(reqs))
	// We MUST have 2 time series as were passed into tsMap.
	gotFromWAL := reqs[0]
	assert.Equal(t, 2, len(gotFromWAL.Timeseries))
	want := &prompb.WriteRequest{
		Timeseries: orderBySampleTimestamp([]prompb.TimeSeries{
			*ts1, *ts2,
		}),
	}

	// Even after sorting timeseries, we need to sort them
	// also by Label to ensure deterministic ordering.
	orderByLabelValue(gotFromWAL)
	orderByLabelValue(want)

	assert.Equal(t, want, gotFromWAL)

	// 3. Finally, ensure that the bytes that were uploaded to the
	// Prometheus Remote Write endpoint are exactly as were saved in the WAL.
	// Read from that same WAL, export to the RWExporter server.
	prwe2, err := NewPrwExporter("test_ns", prweServer.URL, prweServer.Client(), nil, 1, defaultBuildInfo, walConfig)
	assert.Nil(t, err)
	defer prwe2.Shutdown(ctx)

	snappyEncodedBytes := <-uploadedBytesCh
	decodeBuffer := make([]byte, len(snappyEncodedBytes))
	uploadedBytes, err := snappy.Decode(decodeBuffer, snappyEncodedBytes)
	require.Nil(t, err)
	gotFromUpload := new(prompb.WriteRequest)
	err = proto.Unmarshal(uploadedBytes, gotFromUpload)
	assert.Nil(t, err)
	gotFromUpload.Timeseries = orderBySampleTimestamp(gotFromUpload.Timeseries)
	// Even after sorting timeseries, we need to sort them
	// also by Label to ensure deterministic ordering.
	orderByLabelValue(gotFromUpload)

	// 4. Ensure that all the various combinations match up.
	// To ensure a deterministic ordering, sort the TimeSeries by Label Name.
	assert.Equal(t, want, gotFromUpload)
	assert.Equal(t, gotFromWAL, gotFromUpload)
}

func orderByLabelValue(wreq *prompb.WriteRequest) {
	// Sort the timeSeries by their labels.
	type byLabelMessage struct {
		label  *prompb.Label
		sample *prompb.Sample
	}

	for _, timeSeries := range wreq.Timeseries {
		bMsgs := make([]*byLabelMessage, 0, len(wreq.Timeseries)*10)
		for i := range timeSeries.Labels {
			bMsgs = append(bMsgs, &byLabelMessage{
				label:  &timeSeries.Labels[i],
				sample: &timeSeries.Samples[i],
			})
		}
		sort.Slice(bMsgs, func(i, j int) bool {
			return bMsgs[i].label.Value < bMsgs[j].label.Value
		})

		for i := range bMsgs {
			timeSeries.Labels[i] = *bMsgs[i].label
			timeSeries.Samples[i] = *bMsgs[i].sample
		}
	}
}
