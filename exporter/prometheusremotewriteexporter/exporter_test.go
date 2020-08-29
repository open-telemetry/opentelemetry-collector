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
	"sync"
	"testing"

	proto "github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1old"
	"go.opentelemetry.io/collector/internal/dataold"
	"go.opentelemetry.io/collector/internal/dataold/testdataold"
)

// Test_handleScalarMetric checks whether data points within a single scalar metric can be added to a map of
// TimeSeries correctly.
// Test cases are two data point belonging to the same TimeSeries, two data point belonging different TimeSeries,
// and nil data points case.
func Test_handleScalarMetric(t *testing.T) {
	sameTs := map[string]*prompb.TimeSeries{
		// string signature of the data point is the key of the map
		typeMonotonicInt64 + "-__name__-same_ts_int_points_total" + lb1Sig: getTimeSeries(
			getPromLabels(label11, value11, label12, value12, nameStr, "same_ts_int_points_total"),
			getSample(float64(intVal1), msTime1),
			getSample(float64(intVal2), msTime1)),
	}
	differentTs := map[string]*prompb.TimeSeries{
		typeMonotonicDouble + "-__name__-different_ts_double_points_total" + lb1Sig: getTimeSeries(
			getPromLabels(label11, value11, label12, value12, nameStr, "different_ts_double_points_total"),
			getSample(floatVal1, msTime1)),
		typeMonotonicDouble + "-__name__-different_ts_double_points_total" + lb2Sig: getTimeSeries(
			getPromLabels(label21, value21, label22, value22, nameStr, "different_ts_double_points_total"),
			getSample(floatVal2, msTime2)),
	}

	tests := []struct {
		name        string
		m           *otlp.Metric
		returnError bool
		want        map[string]*prompb.TimeSeries
	}{
		{
			"invalid_nil_array",
			&otlp.Metric{
				MetricDescriptor:    getDescriptor("invalid_nil_array", monotonicInt64Comb, validCombinations),
				Int64DataPoints:     nil,
				DoubleDataPoints:    nil,
				HistogramDataPoints: nil,
				SummaryDataPoints:   nil,
			},
			true,
			map[string]*prompb.TimeSeries{},
		},
		{
			"invalid_type_array",
			&otlp.Metric{
				MetricDescriptor:    getDescriptor("invalid_type_array", histogramComb, validCombinations),
				Int64DataPoints:     nil,
				DoubleDataPoints:    nil,
				HistogramDataPoints: nil,
				SummaryDataPoints:   nil,
			},
			true,
			map[string]*prompb.TimeSeries{},
		},
		{
			"same_ts_int_points",
			&otlp.Metric{
				MetricDescriptor: getDescriptor("same_ts_int_points", monotonicInt64Comb, validCombinations),
				Int64DataPoints: []*otlp.Int64DataPoint{
					getIntDataPoint(lbs1, intVal1, time1),
					getIntDataPoint(lbs1, intVal2, time1),
				},
				DoubleDataPoints:    nil,
				HistogramDataPoints: nil,
				SummaryDataPoints:   nil,
			},
			false,
			sameTs,
		},
		{
			"different_ts_double_points",
			&otlp.Metric{
				MetricDescriptor: getDescriptor("different_ts_double_points", monotonicDoubleComb, validCombinations),
				Int64DataPoints:  nil,
				DoubleDataPoints: []*otlp.DoubleDataPoint{
					getDoubleDataPoint(lbs1, floatVal1, time1),
					getDoubleDataPoint(lbs2, floatVal2, time2),
				},
				HistogramDataPoints: nil,
				SummaryDataPoints:   nil,
			},
			false,
			differentTs,
		},
	}
	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tsMap := map[string]*prompb.TimeSeries{}
			prw := &prwExporter{}
			ok := prw.handleScalarMetric(tsMap, tt.m)
			if tt.returnError {
				assert.Error(t, ok)
				return
			}
			assert.Exactly(t, len(tt.want), len(tsMap))
			for k, v := range tsMap {
				require.NotNil(t, tt.want[k])
				assert.ElementsMatch(t, tt.want[k].Labels, v.Labels)
				assert.ElementsMatch(t, tt.want[k].Samples, v.Samples)
			}
		})
	}
}

// Test_newPrwExporter checks that a new exporter instance with non-nil fields is initialized
func Test_newPrwExporter(t *testing.T) {
	config := &Config{
		ExporterSettings:   configmodels.ExporterSettings{},
		TimeoutSettings:    exporterhelper.TimeoutSettings{},
		QueueSettings:      exporterhelper.QueueSettings{},
		RetrySettings:      exporterhelper.RetrySettings{},
		Namespace:          "",
		HTTPClientSettings: confighttp.HTTPClientSettings{Endpoint: ""},
	}
	tests := []struct {
		name        string
		config      *Config
		namespace   string
		endpoint    string
		client      *http.Client
		returnError bool
	}{
		{
			"invalid_URL",
			config,
			"test",
			"invalid URL",
			http.DefaultClient,
			true,
		},
		{
			"nil_client",
			config,
			"test",
			"http://some.url:9411/api/prom/push",
			nil,
			true,
		},
		{
			"success_case",
			config,
			"test",
			"http://some.url:9411/api/prom/push",
			http.DefaultClient,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prwe, err := newPrwExporter(tt.namespace, tt.endpoint, tt.client)
			if tt.returnError {
				assert.Error(t, err)
				return
			}
			require.NotNil(t, prwe)
			assert.NotNil(t, prwe.namespace)
			assert.NotNil(t, prwe.endpointURL)
			assert.NotNil(t, prwe.client)
			assert.NotNil(t, prwe.closeChan)
			assert.NotNil(t, prwe.wg)
		})
	}
}

// Test_shutdown checks after shutdown is called, incoming calls to pushMetrics return error.
func Test_shutdown(t *testing.T) {
	prwe := &prwExporter{
		wg:        new(sync.WaitGroup),
		closeChan: make(chan struct{}),
	}
	wg := new(sync.WaitGroup)
	errChan := make(chan error, 5)
	err := prwe.shutdown(context.Background())
	require.NoError(t, err)
	errChan = make(chan error, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, ok := prwe.pushMetrics(context.Background(),
				pdatautil.MetricsFromOldInternalMetrics(testdataold.GenerateMetricDataEmpty()))
			errChan <- ok
		}()
	}
	wg.Wait()
	close(errChan)
	for ok := range errChan {
		assert.Error(t, ok)
	}
}

//Test whether or not the Server receives the correct TimeSeries.
//Currently considering making this test an iterative for loop of multiple TimeSeries
//Much akin to Test_pushMetrics
func Test_export(t *testing.T) {
	//First we will instantiate a dummy TimeSeries instance to pass into both the export call and compare the http request
	labels := getPromLabels(label11, value11, label12, value12, label21, value21, label22, value22)
	sample1 := getSample(floatVal1, msTime1)
	sample2 := getSample(floatVal2, msTime2)
	ts1 := getTimeSeries(labels, sample1, sample2)
	handleFunc := func(w http.ResponseWriter, r *http.Request, code int) {
		//The following is a handler function that reads the sent httpRequest, unmarshals, and checks if the WriteRequest
		//preserves the TimeSeries data correctly
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		require.NotNil(t, body)
		//Receives the http requests and unzip, unmarshals, and extracts TimeSeries
		assert.Equal(t, "0.1.0", r.Header.Get("X-Prometheus-Remote-Write-Version"))
		assert.Equal(t, "snappy", r.Header.Get("Content-Encoding"))
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
		fmt.Fprintf(w, "error message")
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
				handleFunc(w, r, tt.httpResponseCode)
			}))
			defer server.Close()
			serverURL, uErr := url.Parse(server.URL)
			assert.NoError(t, uErr)
			if !tt.serverUp {
				server.Close()
			}
			err := runExportPipeline(t, ts1, serverURL)
			if tt.returnError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func runExportPipeline(t *testing.T, ts *prompb.TimeSeries, endpoint *url.URL) error {
	//First we will construct a TimeSeries array from the testutils package
	testmap := make(map[string]*prompb.TimeSeries)
	testmap["test"] = ts

	HTTPClient := http.DefaultClient
	//after this, instantiate a CortexExporter with the current HTTP client and endpoint set to passed in endpoint
	prwe, err := newPrwExporter("test", endpoint.String(), HTTPClient)
	if err != nil {
		return err
	}
	err = prwe.export(context.Background(), testmap)
	return err
}

// Test_pushMetrics checks the number of TimeSeries received by server and the number of metrics dropped is the same as
// expected
func Test_pushMetrics(t *testing.T) {
	// fail cases
	noTempBatch := pdatautil.MetricsFromOldInternalMetrics(testdataold.GenerateMetricDataManyMetricsSameResource(10))
	invalidTypeBatch := pdatautil.MetricsFromOldInternalMetrics(testdataold.GenerateMetricDataMetricTypeInvalid())

	invalidTemp := testdataold.GenerateMetricDataManyMetricsSameResource(10)
	setTemporality(&invalidTemp, otlp.MetricDescriptor_INVALID_TEMPORALITY)
	invalidTempBatch := pdatautil.MetricsFromOldInternalMetrics(invalidTemp)

	nilDescBatch := pdatautil.MetricsFromOldInternalMetrics(testdataold.GenerateMetricDataNilMetricDescriptor())
	nilBatch1 := testdataold.GenerateMetricDataManyMetricsSameResource(10)
	nilBatch2 := testdataold.GenerateMetricDataManyMetricsSameResource(10)

	setTemporality(&nilBatch1, otlp.MetricDescriptor_CUMULATIVE)
	setTemporality(&nilBatch2, otlp.MetricDescriptor_CUMULATIVE)
	setDataPointToNil(&nilBatch1, typeMonotonicInt64)
	setType(&nilBatch2, typeMonotonicDouble)

	nilIntDataPointsBatch := pdatautil.MetricsFromOldInternalMetrics(nilBatch1)
	nilDoubleDataPointsBatch := pdatautil.MetricsFromOldInternalMetrics(nilBatch2)

	// Success cases: 10 counter metrics, 2 points in each. Two TimeSeries in total
	batch1 := testdataold.GenerateMetricDataManyMetricsSameResource(10)
	setTemporality(&batch1, otlp.MetricDescriptor_CUMULATIVE)
	scalarBatch := pdatautil.MetricsFromOldInternalMetrics(batch1)

	// Partial Success cases
	batch2 := testdataold.GenerateMetricDataManyMetricsSameResource(10)
	setTemporality(&batch2, otlp.MetricDescriptor_CUMULATIVE)
	failDesc := dataold.MetricDataToOtlp(batch2)[0].InstrumentationLibraryMetrics[0].Metrics[0].GetMetricDescriptor()
	failDesc.Temporality = otlp.MetricDescriptor_INVALID_TEMPORALITY
	partialBatch := pdatautil.MetricsFromOldInternalMetrics(batch2)

	checkFunc := func(t *testing.T, r *http.Request, expected int) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}

		buf := make([]byte, len(body))
		dest, err := snappy.Decode(buf, body)
		assert.Equal(t, "0.1.0", r.Header.Get("x-prometheus-remote-write-version"))
		assert.Equal(t, "snappy", r.Header.Get("content-encoding"))
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
			pdatautil.MetricCount(invalidTypeBatch),
			true,
		},
		{
			"invalid_temporality_case",
			&invalidTempBatch,
			nil,
			0,
			http.StatusAccepted,
			pdatautil.MetricCount(invalidTempBatch),
			true,
		},
		{
			"nil_desc_case",
			&nilDescBatch,
			nil,
			0,
			http.StatusAccepted,
			pdatautil.MetricCount(nilDescBatch),
			true,
		},
		{
			"nil_int_point_case",
			&nilIntDataPointsBatch,
			nil,
			0,
			http.StatusAccepted,
			pdatautil.MetricCount(nilIntDataPointsBatch),
			true,
		},
		{
			"nil_double_point_case",
			&nilDoubleDataPointsBatch,
			nil,
			0,
			http.StatusAccepted,
			pdatautil.MetricCount(nilDoubleDataPointsBatch),
			true,
		},
		{
			"no_temp_case",
			&noTempBatch,
			nil,
			0,
			http.StatusAccepted,
			pdatautil.MetricCount(noTempBatch),
			true,
		},
		{
			"http_error_case",
			&noTempBatch,
			nil,
			0,
			http.StatusForbidden,
			pdatautil.MetricCount(noTempBatch),
			true,
		},
		{
			"scalar_case",
			&scalarBatch,
			checkFunc,
			2,
			http.StatusAccepted,
			0,
			false,
		},
		{
			"partial_success_case",
			&partialBatch,
			checkFunc,
			2,
			http.StatusAccepted,
			1,
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
			prwe, nErr := newPrwExporter(config.Namespace, serverURL.String(), c)
			require.NoError(t, nErr)
			numDroppedTimeSeries, err := prwe.pushMetrics(context.Background(), *tt.md)
			assert.Equal(t, tt.numDroppedTimeSeries, numDroppedTimeSeries)
			if tt.returnErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
