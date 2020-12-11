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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/internal/version"
)

// Test_ NewPrwExporter checks that a new exporter instance with non-nil fields is initialized
func Test_NewPrwExporter(t *testing.T) {
	config := &Config{
		ExporterSettings:   configmodels.ExporterSettings{},
		TimeoutSettings:    exporterhelper.TimeoutSettings{},
		QueueSettings:      exporterhelper.QueueSettings{},
		RetrySettings:      exporterhelper.RetrySettings{},
		Namespace:          "",
		ExternalLabels:     map[string]string{},
		HTTPClientSettings: confighttp.HTTPClientSettings{Endpoint: ""},
	}
	tests := []struct {
		name           string
		config         *Config
		namespace      string
		endpoint       string
		externalLabels map[string]string
		client         *http.Client
		returnError    bool
	}{
		{
			"invalid_URL",
			config,
			"test",
			"invalid URL",
			map[string]string{"Key1": "Val1"},
			http.DefaultClient,
			true,
		},
		{
			"nil_client",
			config,
			"test",
			"http://some.url:9411/api/prom/push",
			map[string]string{"Key1": "Val1"},
			nil,
			true,
		},
		{
			"invalid_labels_case",
			config,
			"test",
			"http://some.url:9411/api/prom/push",
			map[string]string{"Key1": ""},
			http.DefaultClient,
			true,
		},
		{
			"success_case",
			config,
			"test",
			"http://some.url:9411/api/prom/push",
			map[string]string{"Key1": "Val1"},
			http.DefaultClient,
			false,
		},
		{
			"success_case_no_labels",
			config,
			"test",
			"http://some.url:9411/api/prom/push",
			map[string]string{},
			http.DefaultClient,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prwe, err := NewPrwExporter(tt.namespace, tt.endpoint, tt.client, tt.externalLabels)
			if tt.returnError {
				assert.Error(t, err)
				return
			}
			require.NotNil(t, prwe)
			assert.NotNil(t, prwe.namespace)
			assert.NotNil(t, prwe.endpointURL)
			assert.NotNil(t, prwe.externalLabels)
			assert.NotNil(t, prwe.client)
			assert.NotNil(t, prwe.closeChan)
			assert.NotNil(t, prwe.wg)
		})
	}
}

// Test_Shutdown checks after Shutdown is called, incoming calls to PushMetrics return error.
func Test_Shutdown(t *testing.T) {
	prwe := &PrwExporter{
		wg:        new(sync.WaitGroup),
		closeChan: make(chan struct{}),
	}
	wg := new(sync.WaitGroup)
	errChan := make(chan error, 5)
	err := prwe.Shutdown(context.Background())
	require.NoError(t, err)
	errChan = make(chan error, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, ok := prwe.PushMetrics(context.Background(), testdata.GenerateMetricsEmpty())
			errChan <- ok
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
		// The following is a handler function that reads the sent httpRequest, unmarshals, and checks if the WriteRequest
		// preserves the TimeSeries data correctly
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		require.NotNil(t, body)
		// Receives the http requests and unzip, unmarshals, and extracts TimeSeries
		assert.Equal(t, "0.1.0", r.Header.Get("X-Prometheus-Remote-Write-Version"))
		assert.Equal(t, "snappy", r.Header.Get("Content-Encoding"))
		assert.Equal(t, "OpenTelemetry-Collector/"+version.Version, r.Header.Get("User-Agent"))
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
		name             string
		ts               prompb.TimeSeries
		serverUp         bool
		httpResponseCode int
		returnError      bool
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
			if tt.returnError {
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

	HTTPClient := http.DefaultClient
	// after this, instantiate a CortexExporter with the current HTTP client and endpoint set to passed in endpoint
	prwe, err := NewPrwExporter("test", endpoint.String(), HTTPClient, map[string]string{})
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	errs = append(errs, prwe.export(context.Background(), testmap)...)
	return errs
}

// Test_PushMetrics checks the number of TimeSeries received by server and the number of metrics dropped is the same as
// expected
func Test_PushMetrics(t *testing.T) {

	invalidTypeBatch := testdata.GenerateMetricsMetricTypeInvalid()

	// success cases
	intSumBatch := testdata.GenerateMetricsManyMetricsSameResource(10)

	doubleSumMetric := []*otlp.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlp.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlp.Metric{
						validMetrics1[validDoubleSum],
						validMetrics2[validDoubleSum],
					},
				},
			},
		},
	}
	doubleSumBatch := pdata.MetricsFromOtlp(doubleSumMetric)

	intGaugeMetric := []*otlp.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlp.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlp.Metric{
						validMetrics1[validIntGauge],
						validMetrics2[validIntGauge],
					},
				},
			},
		},
	}
	intGaugeBatch := pdata.MetricsFromOtlp(intGaugeMetric)

	doubleGaugeMetric := []*otlp.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlp.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlp.Metric{
						validMetrics1[validDoubleGauge],
						validMetrics2[validDoubleGauge],
					},
				},
			},
		},
	}
	doubleGaugeBatch := pdata.MetricsFromOtlp(doubleGaugeMetric)

	intHistogramMetric := []*otlp.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlp.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlp.Metric{
						validMetrics1[validIntHistogram],
						validMetrics2[validIntHistogram],
					},
				},
			},
		},
	}
	intHistogramBatch := pdata.MetricsFromOtlp(intHistogramMetric)

	doubleHistogramMetric := []*otlp.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlp.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlp.Metric{
						validMetrics1[validDoubleHistogram],
						validMetrics2[validDoubleHistogram],
					},
				},
			},
		},
	}
	doubleHistogramBatch := pdata.MetricsFromOtlp(doubleHistogramMetric)

	doubleSummaryMetric := []*otlp.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlp.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlp.Metric{
						validMetrics1[validDoubleSummary],
						validMetrics2[validDoubleSummary],
					},
				},
			},
		},
	}
	doubleSummaryBatch := pdata.MetricsFromOtlp(doubleSummaryMetric)

	// len(BucketCount) > len(ExplicitBounds)
	unmatchedBoundBucketIntHistMetric := []*otlp.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlp.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlp.Metric{
						validMetrics2[unmatchedBoundBucketIntHist],
					},
				},
			},
		},
	}
	unmatchedBoundBucketIntHistBatch := pdata.MetricsFromOtlp(unmatchedBoundBucketIntHistMetric)

	unmatchedBoundBucketDoubleHistMetric := []*otlp.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlp.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlp.Metric{
						validMetrics2[unmatchedBoundBucketDoubleHist],
					},
				},
			},
		},
	}
	unmatchedBoundBucketDoubleHistBatch := pdata.MetricsFromOtlp(unmatchedBoundBucketDoubleHistMetric)

	// fail cases
	nilDataPointIntGaugeMetric := []*otlp.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlp.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlp.Metric{
						errorMetrics[nilDataPointIntGauge],
					},
				},
			},
		},
	}
	nilDataPointIntGaugeBatch := pdata.MetricsFromOtlp(nilDataPointIntGaugeMetric)

	nilDataPointDoubleGaugeMetric := []*otlp.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlp.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlp.Metric{
						errorMetrics[nilDataPointDoubleGauge],
					},
				},
			},
		},
	}
	nilDataPointDoubleGaugeBatch := pdata.MetricsFromOtlp(nilDataPointDoubleGaugeMetric)

	nilDataPointIntSumMetric := []*otlp.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlp.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlp.Metric{
						errorMetrics[nilDataPointIntSum],
					},
				},
			},
		},
	}
	nilDataPointIntSumBatch := pdata.MetricsFromOtlp(nilDataPointIntSumMetric)

	nilDataPointDoubleSumMetric := []*otlp.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlp.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlp.Metric{
						errorMetrics[nilDataPointDoubleSum],
					},
				},
			},
		},
	}
	nilDataPointDoubleSumBatch := pdata.MetricsFromOtlp(nilDataPointDoubleSumMetric)

	nilDataPointIntHistogramMetric := []*otlp.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlp.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlp.Metric{
						errorMetrics[nilDataPointIntHistogram],
					},
				},
			},
		},
	}
	nilDataPointIntHistogramBatch := pdata.MetricsFromOtlp(nilDataPointIntHistogramMetric)

	nilDataPointDoubleHistogramMetric := []*otlp.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlp.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlp.Metric{
						errorMetrics[nilDataPointDoubleHistogram],
					},
				},
			},
		},
	}
	nilDataPointDoubleHistogramBatch := pdata.MetricsFromOtlp(nilDataPointDoubleHistogramMetric)

	nilDataPointDoubleSummaryMetric := []*otlp.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlp.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlp.Metric{
						errorMetrics[nilDataPointDoubleSummary],
					},
				},
			},
		},
	}
	nilDataPointDoubleSummaryBatch := pdata.MetricsFromOtlp(nilDataPointDoubleSummaryMetric)

	checkFunc := func(t *testing.T, r *http.Request, expected int) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}

		buf := make([]byte, len(body))
		dest, err := snappy.Decode(buf, body)
		assert.Equal(t, "0.1.0", r.Header.Get("x-prometheus-remote-write-version"))
		assert.Equal(t, "snappy", r.Header.Get("content-encoding"))
		assert.Equal(t, "OpenTelemetry-Collector/"+version.Version, r.Header.Get("user-agent"))
		assert.NotNil(t, r.Header.Get("tenant-id"))
		require.NoError(t, err)
		wr := &prompb.WriteRequest{}
		ok := proto.Unmarshal(dest, wr)
		require.Nil(t, ok)
		assert.EqualValues(t, expected, len(wr.Timeseries))
	}

	tests := []struct {
		name                 string
		md                   *pdata.Metrics
		reqTestFunc          func(t *testing.T, r *http.Request, expected int)
		expectedTimeSeries   int
		httpResponseCode     int
		numDroppedTimeSeries int
		returnErr            bool
	}{
		{
			"invalid_type_case",
			&invalidTypeBatch,
			nil,
			0,
			http.StatusAccepted,
			invalidTypeBatch.MetricCount(),
			true,
		},
		{
			"intSum_case",
			&intSumBatch,
			checkFunc,
			2,
			http.StatusAccepted,
			0,
			false,
		},
		{
			"doubleSum_case",
			&doubleSumBatch,
			checkFunc,
			2,
			http.StatusAccepted,
			0,
			false,
		},
		{
			"doubleGauge_case",
			&doubleGaugeBatch,
			checkFunc,
			2,
			http.StatusAccepted,
			0,
			false,
		},
		{
			"intGauge_case",
			&intGaugeBatch,
			checkFunc,
			2,
			http.StatusAccepted,
			0,
			false,
		},
		{
			"intHistogram_case",
			&intHistogramBatch,
			checkFunc,
			12,
			http.StatusAccepted,
			0,
			false,
		},
		{
			"doubleHistogram_case",
			&doubleHistogramBatch,
			checkFunc,
			12,
			http.StatusAccepted,
			0,
			false,
		},
		{
			"doubleSummary_case",
			&doubleSummaryBatch,
			checkFunc,
			10,
			http.StatusAccepted,
			0,
			false,
		},
		{
			"unmatchedBoundBucketIntHist_case",
			&unmatchedBoundBucketIntHistBatch,
			checkFunc,
			5,
			http.StatusAccepted,
			0,
			false,
		},
		{
			"unmatchedBoundBucketDoubleHist_case",
			&unmatchedBoundBucketDoubleHistBatch,
			checkFunc,
			5,
			http.StatusAccepted,
			0,
			false,
		},
		{
			"5xx_case",
			&unmatchedBoundBucketDoubleHistBatch,
			checkFunc,
			5,
			http.StatusServiceUnavailable,
			1,
			true,
		},
		{
			"nilDataPointDoubleGauge_case",
			&nilDataPointDoubleGaugeBatch,
			checkFunc,
			0,
			http.StatusAccepted,
			nilDataPointDoubleGaugeBatch.MetricCount(),
			true,
		},
		{
			"nilDataPointIntGauge_case",
			&nilDataPointIntGaugeBatch,
			checkFunc,
			0,
			http.StatusAccepted,
			nilDataPointIntGaugeBatch.MetricCount(),
			true,
		},
		{
			"nilDataPointDoubleSum_case",
			&nilDataPointDoubleSumBatch,
			checkFunc,
			0,
			http.StatusAccepted,
			nilDataPointDoubleSumBatch.MetricCount(),
			true,
		},
		{
			"nilDataPointIntSum_case",
			&nilDataPointIntSumBatch,
			checkFunc,
			0,
			http.StatusAccepted,
			nilDataPointIntSumBatch.MetricCount(),
			true,
		},
		{
			"nilDataPointDoubleHistogram_case",
			&nilDataPointDoubleHistogramBatch,
			checkFunc,
			0,
			http.StatusAccepted,
			nilDataPointDoubleHistogramBatch.MetricCount(),
			true,
		},
		{
			"nilDataPointIntHistogram_case",
			&nilDataPointIntHistogramBatch,
			checkFunc,
			0,
			http.StatusAccepted,
			nilDataPointIntHistogramBatch.MetricCount(),
			true,
		},
		{
			"nilDataPointDoubleSummary_case",
			&nilDataPointDoubleSummaryBatch,
			checkFunc,
			0,
			http.StatusAccepted,
			nilDataPointDoubleSummaryBatch.MetricCount(),
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

			serverURL, uErr := url.Parse(server.URL)
			assert.NoError(t, uErr)

			config := &Config{
				ExporterSettings: configmodels.ExporterSettings{
					TypeVal: "prometheusremotewrite",
					NameVal: "prometheusremotewrite",
				},
				Namespace: "",
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Endpoint: "http://some.url:9411/api/prom/push",
					// We almost read 0 bytes, so no need to tune ReadBufferSize.
					ReadBufferSize:  0,
					WriteBufferSize: 512 * 1024,
				},
			}
			assert.NotNil(t, config)
			// c, err := config.HTTPClientSettings.ToClient()
			// assert.Nil(t, err)
			c := http.DefaultClient
			prwe, nErr := NewPrwExporter(config.Namespace, serverURL.String(), c, map[string]string{})
			require.NoError(t, nErr)
			numDroppedTimeSeries, err := prwe.PushMetrics(context.Background(), *tt.md)
			assert.Equal(t, tt.numDroppedTimeSeries, numDroppedTimeSeries)
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
		name           string
		inputLabels    map[string]string
		expectedLabels map[string]string
		returnError    bool
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
			if tt.returnError {
				assert.Error(t, err)
				return
			}
			assert.EqualValues(t, tt.expectedLabels, newLabels)
			assert.NoError(t, err)
		})
	}
}
