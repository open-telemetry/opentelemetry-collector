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

package prometheusremotewriteexporter

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configparser"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/model/pdata"
)

// Test_NewPRWExporter checks that a new exporter instance with non-nil fields is initialized
func Test_NewPRWExporter(t *testing.T) {
	cfg := &Config{
		ExporterSettings:   config.NewExporterSettings(config.NewID(typeStr)),
		TimeoutSettings:    exporterhelper.TimeoutSettings{},
		RetrySettings:      exporterhelper.RetrySettings{},
		Namespace:          "",
		ExternalLabels:     map[string]string{},
		HTTPClientSettings: confighttp.HTTPClientSettings{Endpoint: ""},
	}
	buildInfo := component.BuildInfo{
		Description: "OpenTelemetry Collector",
		Version:     "1.0",
	}

	tests := []struct {
		name                string
		config              *Config
		namespace           string
		endpoint            string
		concurrency         int
		externalLabels      map[string]string
		returnErrorOnCreate bool
		buildInfo           component.BuildInfo
	}{
		{
			name:                "invalid_URL",
			config:              cfg,
			namespace:           "test",
			endpoint:            "invalid URL",
			concurrency:         5,
			externalLabels:      map[string]string{"Key1": "Val1"},
			returnErrorOnCreate: true,
			buildInfo:           buildInfo,
		},
		{
			name:                "invalid_labels_case",
			config:              cfg,
			namespace:           "test",
			endpoint:            "http://some.url:9411/api/prom/push",
			concurrency:         5,
			externalLabels:      map[string]string{"Key1": ""},
			returnErrorOnCreate: true,
			buildInfo:           buildInfo,
		},
		{
			name:           "success_case",
			config:         cfg,
			namespace:      "test",
			endpoint:       "http://some.url:9411/api/prom/push",
			concurrency:    5,
			externalLabels: map[string]string{"Key1": "Val1"},
			buildInfo:      buildInfo,
		},
		{
			name:           "success_case_no_labels",
			config:         cfg,
			namespace:      "test",
			endpoint:       "http://some.url:9411/api/prom/push",
			concurrency:    5,
			externalLabels: map[string]string{},
			buildInfo:      buildInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.HTTPClientSettings.Endpoint = tt.endpoint
			cfg.ExternalLabels = tt.externalLabels
			cfg.Namespace = tt.namespace
			cfg.RemoteWriteQueue.NumConsumers = 1
			prwe, err := NewPRWExporter(cfg, tt.buildInfo)

			if tt.returnErrorOnCreate {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			require.NotNil(t, prwe)
			assert.NotNil(t, prwe.namespace)
			assert.NotNil(t, prwe.endpointURL)
			assert.NotNil(t, prwe.externalLabels)
			assert.NotNil(t, prwe.closeChan)
			assert.NotNil(t, prwe.wg)
			assert.NotNil(t, prwe.userAgentHeader)
			assert.NotNil(t, prwe.clientSettings)
		})
	}
}

// Test_Start checks if the client is properly created as expected.
func Test_Start(t *testing.T) {
	cfg := &Config{
		ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
		TimeoutSettings:  exporterhelper.TimeoutSettings{},
		RetrySettings:    exporterhelper.RetrySettings{},
		Namespace:        "",
		ExternalLabels:   map[string]string{},
	}
	buildInfo := component.BuildInfo{
		Description: "OpenTelemetry Collector",
		Version:     "1.0",
	}
	tests := []struct {
		name                 string
		config               *Config
		namespace            string
		concurrency          int
		externalLabels       map[string]string
		returnErrorOnStartUp bool
		buildInfo            component.BuildInfo
		endpoint             string
		clientSettings       confighttp.HTTPClientSettings
	}{
		{
			name:           "success_case",
			config:         cfg,
			namespace:      "test",
			concurrency:    5,
			externalLabels: map[string]string{"Key1": "Val1"},
			buildInfo:      buildInfo,
			clientSettings: confighttp.HTTPClientSettings{Endpoint: "https://some.url:9411/api/prom/push"},
		},
		{
			name:                 "invalid_tls",
			config:               cfg,
			namespace:            "test",
			concurrency:          5,
			externalLabels:       map[string]string{"Key1": "Val1"},
			buildInfo:            buildInfo,
			returnErrorOnStartUp: true,
			clientSettings: confighttp.HTTPClientSettings{
				Endpoint: "https://some.url:9411/api/prom/push",
				TLSSetting: configtls.TLSClientSetting{
					TLSSetting: configtls.TLSSetting{
						CAFile:   "non-existent file",
						CertFile: "",
						KeyFile:  "",
					},
					Insecure:   false,
					ServerName: "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.ExternalLabels = tt.externalLabels
			cfg.Namespace = tt.namespace
			cfg.RemoteWriteQueue.NumConsumers = 1
			cfg.HTTPClientSettings = tt.clientSettings

			prwe, err := NewPRWExporter(cfg, tt.buildInfo)
			assert.NoError(t, err)
			assert.NotNil(t, prwe)

			err = prwe.Start(context.Background(), componenttest.NewNopHost())
			if tt.returnErrorOnStartUp {
				assert.Error(t, err)
				return
			}
			assert.NotNil(t, prwe.client)
		})
	}
}

// Test_Shutdown checks after Shutdown is called, incoming calls to PushMetrics return error.
func Test_Shutdown(t *testing.T) {
	prwe := &PRWExporter{
		wg:        new(sync.WaitGroup),
		closeChan: make(chan struct{}),
	}
	wg := new(sync.WaitGroup)
	err := prwe.Shutdown(context.Background())
	require.NoError(t, err)
	errChan := make(chan error, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errChan <- prwe.PushMetrics(context.Background(), pdata.NewMetrics())
		}()
	}
	wg.Wait()
	close(errChan)
	for ok := range errChan {
		assert.Error(t, ok)
	}
}

// Test whether or not the Server receives the correct TimeSeries.
// Currently considering making this test an iterative for loop of multiple TimeSeries much akin to Test_PushMetrics
func Test_export(t *testing.T) {
	// First we will instantiate a dummy TimeSeries instance to pass into both the export call and compare the http request
	labels := getPromLabels(label11, value11, label12, value12, label21, value21, label22, value22)
	sample1 := getSample(floatVal1, msTime1)
	sample2 := getSample(floatVal2, msTime2)
	ts1 := getTimeSeries(labels, sample1, sample2)
	handleFunc := func(w http.ResponseWriter, r *http.Request, code int) {
		// The following is a handler function that reads the sent httpRequest, unmarshal, and checks if the WriteRequest
		// preserves the TimeSeries data correctly
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		require.NotNil(t, body)
		// Receives the http requests and unzip, unmarshalls, and extracts TimeSeries
		assert.Equal(t, "0.1.0", r.Header.Get("X-Prometheus-Remote-Write-Version"))
		assert.Equal(t, "snappy", r.Header.Get("Content-Encoding"))
		assert.Equal(t, "opentelemetry-collector/1.0", r.Header.Get("User-Agent"))
		writeReq := &prompb.WriteRequest{}
		unzipped := []byte{}

		dest, err := snappy.Decode(unzipped, body)
		require.NoError(t, err)

		ok := proto.Unmarshal(dest, writeReq)
		require.NoError(t, ok)

		assert.EqualValues(t, 1, len(writeReq.Timeseries))
		require.NotNil(t, writeReq.GetTimeseries())
		assert.Equal(t, *ts1, writeReq.GetTimeseries()[0])
		w.WriteHeader(code)
	}

	// Create in test table format to check if different HTTP response codes or server errors
	// are properly identified
	tests := []struct {
		name                string
		ts                  prompb.TimeSeries
		serverUp            bool
		httpResponseCode    int
		returnErrorOnCreate bool
	}{
		{"success_case",
			*ts1,
			true,
			http.StatusAccepted,
			false,
		},
		{
			"server_no_response_case",
			*ts1,
			false,
			http.StatusAccepted,
			true,
		}, {
			"error_status_code_case",
			*ts1,
			true,
			http.StatusForbidden,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if handleFunc != nil {
					handleFunc(w, r, tt.httpResponseCode)
				}
			}))
			defer server.Close()
			serverURL, uErr := url.Parse(server.URL)
			assert.NoError(t, uErr)
			if !tt.serverUp {
				server.Close()
			}
			errs := runExportPipeline(ts1, serverURL)
			if tt.returnErrorOnCreate {
				assert.Error(t, errs[0])
				return
			}
			assert.Len(t, errs, 0)
		})
	}
}

func runExportPipeline(ts *prompb.TimeSeries, endpoint *url.URL) []error {
	var errs []error

	// First we will construct a TimeSeries array from the testutils package
	testmap := make(map[string]*prompb.TimeSeries)
	testmap["test"] = ts

	cfg := createDefaultConfig().(*Config)
	cfg.HTTPClientSettings.Endpoint = endpoint.String()
	cfg.RemoteWriteQueue.NumConsumers = 1

	buildInfo := component.BuildInfo{
		Description: "OpenTelemetry Collector",
		Version:     "1.0",
	}
	// after this, instantiate a CortexExporter with the current HTTP client and endpoint set to passed in endpoint
	prwe, err := NewPRWExporter(cfg, buildInfo)
	if err != nil {
		errs = append(errs, err)
		return errs
	}

	if err = prwe.Start(context.Background(), componenttest.NewNopHost()); err != nil {
		errs = append(errs, err)
		return errs
	}

	errs = append(errs, prwe.handleExport(context.Background(), testmap)...)
	return errs
}

// Test_PushMetrics checks the number of TimeSeries received by server and the number of metrics dropped is the same as
// expected
func Test_PushMetrics(t *testing.T) {

	invalidTypeBatch := testdata.GenerateMetricsMetricTypeInvalid()

	// success cases
	intSumBatch := testdata.GenerateMetricsManyMetricsSameResource(10)

	sumBatch := getMetricsFromMetricList(validMetrics1[validSum], validMetrics2[validSum])

	intGaugeBatch := getMetricsFromMetricList(validMetrics1[validIntGauge], validMetrics2[validIntGauge])

	doubleGaugeBatch := getMetricsFromMetricList(validMetrics1[validDoubleGauge], validMetrics2[validDoubleGauge])

	histogramBatch := getMetricsFromMetricList(validMetrics1[validHistogram], validMetrics2[validHistogram])

	summaryBatch := getMetricsFromMetricList(validMetrics1[validSummary], validMetrics2[validSummary])

	// len(BucketCount) > len(ExplicitBounds)
	unmatchedBoundBucketHistBatch := getMetricsFromMetricList(validMetrics2[unmatchedBoundBucketHist])

	// fail cases
	emptyDoubleGaugeBatch := getMetricsFromMetricList(invalidMetrics[emptyGauge])

	emptyCumulativeSumBatch := getMetricsFromMetricList(invalidMetrics[emptyCumulativeSum])

	emptyCumulativeHistogramBatch := getMetricsFromMetricList(invalidMetrics[emptyCumulativeHistogram])

	emptySummaryBatch := getMetricsFromMetricList(invalidMetrics[emptySummary])

	checkFunc := func(t *testing.T, r *http.Request, expected int) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}

		buf := make([]byte, len(body))
		dest, err := snappy.Decode(buf, body)
		assert.Equal(t, "0.1.0", r.Header.Get("x-prometheus-remote-write-version"))
		assert.Equal(t, "snappy", r.Header.Get("content-encoding"))
		assert.Equal(t, "opentelemetry-collector/1.0", r.Header.Get("User-Agent"))
		assert.NotNil(t, r.Header.Get("tenant-id"))
		require.NoError(t, err)
		wr := &prompb.WriteRequest{}
		ok := proto.Unmarshal(dest, wr)
		require.Nil(t, ok)
		assert.EqualValues(t, expected, len(wr.Timeseries))
	}

	tests := []struct {
		name               string
		md                 *pdata.Metrics
		reqTestFunc        func(t *testing.T, r *http.Request, expected int)
		expectedTimeSeries int
		httpResponseCode   int
		returnErr          bool
	}{
		{
			"invalid_type_case",
			&invalidTypeBatch,
			nil,
			0,
			http.StatusAccepted,
			true,
		},
		{
			"intSum_case",
			&intSumBatch,
			checkFunc,
			2,
			http.StatusAccepted,
			false,
		},
		{
			"doubleSum_case",
			&sumBatch,
			checkFunc,
			2,
			http.StatusAccepted,
			false,
		},
		{
			"doubleGauge_case",
			&doubleGaugeBatch,
			checkFunc,
			2,
			http.StatusAccepted,
			false,
		},
		{
			"intGauge_case",
			&intGaugeBatch,
			checkFunc,
			2,
			http.StatusAccepted,
			false,
		},
		{
			"histogram_case",
			&histogramBatch,
			checkFunc,
			12,
			http.StatusAccepted,
			false,
		},
		{
			"summary_case",
			&summaryBatch,
			checkFunc,
			10,
			http.StatusAccepted,
			false,
		},
		{
			"unmatchedBoundBucketHist_case",
			&unmatchedBoundBucketHistBatch,
			checkFunc,
			5,
			http.StatusAccepted,
			false,
		},
		{
			"5xx_case",
			&unmatchedBoundBucketHistBatch,
			checkFunc,
			5,
			http.StatusServiceUnavailable,
			true,
		},
		{
			"emptyGauge_case",
			&emptyDoubleGaugeBatch,
			checkFunc,
			0,
			http.StatusAccepted,
			true,
		},
		{
			"emptyCumulativeSum_case",
			&emptyCumulativeSumBatch,
			checkFunc,
			0,
			http.StatusAccepted,
			true,
		},
		{
			"emptyCumulativeHistogram_case",
			&emptyCumulativeHistogramBatch,
			checkFunc,
			0,
			http.StatusAccepted,
			true,
		},
		{
			"emptySummary_case",
			&emptySummaryBatch,
			checkFunc,
			0,
			http.StatusAccepted,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.reqTestFunc != nil {
					tt.reqTestFunc(t, r, tt.expectedTimeSeries)
				}
				w.WriteHeader(tt.httpResponseCode)
			}))

			defer server.Close()

			cfg := &Config{
				ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
				Namespace:        "",
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Endpoint: server.URL,
					// We almost read 0 bytes, so no need to tune ReadBufferSize.
					ReadBufferSize:  0,
					WriteBufferSize: 512 * 1024,
				},
				RemoteWriteQueue: RemoteWriteQueue{NumConsumers: 5},
			}
			assert.NotNil(t, cfg)
			// c, err := config.HTTPClientSettings.ToClient()
			// assert.Nil(t, err)
			buildInfo := component.BuildInfo{
				Description: "OpenTelemetry Collector",
				Version:     "1.0",
			}
			prwe, nErr := NewPRWExporter(cfg, buildInfo)
			require.NoError(t, nErr)
			require.NoError(t, prwe.Start(context.Background(), componenttest.NewNopHost()))
			err := prwe.PushMetrics(context.Background(), *tt.md)
			if tt.returnErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func Test_validateAndSanitizeExternalLabels(t *testing.T) {
	tests := []struct {
		name                string
		inputLabels         map[string]string
		expectedLabels      map[string]string
		returnErrorOnCreate bool
	}{
		{"success_case_no_labels",
			map[string]string{},
			map[string]string{},
			false,
		},
		{"success_case_with_labels",
			map[string]string{"key1": "val1"},
			map[string]string{"key1": "val1"},
			false,
		},
		{"success_case_2_with_labels",
			map[string]string{"__key1__": "val1"},
			map[string]string{"__key1__": "val1"},
			false,
		},
		{"success_case_with_sanitized_labels",
			map[string]string{"__key1.key__": "val1"},
			map[string]string{"__key1_key__": "val1"},
			false,
		},
		{"fail_case_empty_label",
			map[string]string{"": "val1"},
			map[string]string{},
			true,
		},
	}
	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newLabels, err := validateAndSanitizeExternalLabels(tt.inputLabels)
			if tt.returnErrorOnCreate {
				assert.Error(t, err)
				return
			}
			assert.EqualValues(t, tt.expectedLabels, newLabels)
			assert.NoError(t, err)
		})
	}
}

// Ensures that when we attach the Write-Ahead-Log(WAL) to the exporter,
// that it successfully writes the serialized prompb.WriteRequests to the WAL,
// and that we can retrieve those exact requests back from the WAL, when the
// exporter starts up once again, that it picks up where it left off.
func TestWALOnExporterRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("This test could run for long")
	}

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

	// 2. Create the WAL configuration, create the
	// exporter and export some time series!
	tempDir := t.TempDir()
	yamlConfig := fmt.Sprintf(`
namespace: "test_ns"
endpoint: %q
remote_write_queue:
  num_consumers: 1

wal:
    directory:           %s
    truncate_frequency:  60us
    buffer_size:          1
        `, prweServer.URL, tempDir)

	parser, err := configparser.NewParserFromBuffer(strings.NewReader(yamlConfig))
	require.Nil(t, err)

	fullConfig := new(Config)
	if err = parser.UnmarshalExact(fullConfig); err != nil {
		t.Fatalf("Failed to parse config from YAML: %v", err)
	}
	walConfig := fullConfig.WAL
	require.NotNil(t, walConfig)
	require.Equal(t, walConfig.TruncateFrequency, 60*time.Microsecond)
	require.Equal(t, walConfig.Directory, tempDir)

	var defaultBuildInfo = component.BuildInfo{
		Description: "OpenTelemetry Collector",
		Version:     "1.0",
	}
	prwe, perr := NewPRWExporter(fullConfig, defaultBuildInfo)
	assert.Nil(t, perr)

	nopHost := componenttest.NewNopHost()
	ctx := context.Background()
	require.Nil(t, prwe.Start(ctx, nopHost))
	defer prwe.Shutdown(ctx)
	defer close(exiting)
	require.NotNil(t, prwe.wal)

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
	errs := prwe.handleExport(ctx, tsMap)
	assert.Nil(t, errs)
	// Shutdown after we've written to the WAL. This ensures that our
	// exported data in-flight will flushed flushed to the WAL before exiting.
	prwe.Shutdown(ctx)

	// 3. Let's now read back all of the WAL records and ensure
	// that all the prompb.WriteRequest values exist as we sent them.
	wal, _, werr := walConfig.createWAL()
	assert.Nil(t, werr)
	assert.NotNil(t, wal)
	defer wal.Close()

	// Read all the indices.
	firstIndex, ierr := wal.FirstIndex()
	assert.Nil(t, ierr)
	lastIndex, ierr := wal.LastIndex()
	assert.Nil(t, ierr)

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
	gotFromWAL.Timeseries = orderBySampleTimestamp(gotFromWAL.Timeseries)
	orderByLabelValue(want)

	assert.Equal(t, want, gotFromWAL)

	// 4. Finally, ensure that the bytes that were uploaded to the
	// Prometheus Remote Write endpoint are exactly as were saved in the WAL.
	// Read from that same WAL, export to the RWExporter server.
	prwe2, err := NewPRWExporter(fullConfig, defaultBuildInfo)
	assert.Nil(t, err)
	require.Nil(t, prwe2.Start(ctx, nopHost))
	defer prwe2.Shutdown(ctx)
	require.NotNil(t, prwe2.wal)

	snappyEncodedBytes := <-uploadedBytesCh
	decodeBuffer := make([]byte, len(snappyEncodedBytes))
	uploadedBytes, derr := snappy.Decode(decodeBuffer, snappyEncodedBytes)
	require.Nil(t, derr)
	gotFromUpload := new(prompb.WriteRequest)
	uerr := proto.Unmarshal(uploadedBytes, gotFromUpload)
	assert.Nil(t, uerr)
	gotFromUpload.Timeseries = orderBySampleTimestamp(gotFromUpload.Timeseries)
	// Even after sorting timeseries, we need to sort them
	// also by Label to ensure deterministic ordering.
	orderByLabelValue(gotFromUpload)

	// 4.1. Ensure that all the various combinations match up.
	// To ensure a deterministic ordering, sort the TimeSeries by Label Name.
	assert.Equal(t, want, gotFromUpload)
	assert.Equal(t, gotFromWAL, gotFromUpload)
}
