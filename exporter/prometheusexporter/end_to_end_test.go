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

package prometheusexporter

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
	"time"

	promconfig "github.com/prometheus/prometheus/config"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/receiver/prometheusreceiver"
)

func TestEndToEndSummarySupport(t *testing.T) {
	if testing.Short() {
		t.Skip("This test can take a couple of seconds")
	}

	// 1. Create the Prometheus scrape endpoint.
	waitForScrape := make(chan bool, 1)
	shutdown := make(chan bool, 1)
	dropWizardServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		select {
		case <-shutdown:
			return
		case waitForScrape <- true:
			// Serve back the metrics as if they were from DropWizard.
			_, err := rw.Write([]byte(dropWizardResponse))
			require.NoError(t, err)
		}
	}))
	defer dropWizardServer.Close()
	defer close(shutdown)

	srvURL, err := url.Parse(dropWizardServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Create the Prometheus metrics exporter that'll receive and verify the metrics produced.
	exporterCfg := &Config{
		ExporterSettings: config.NewExporterSettings(config.NewID(typeStr)),
		Namespace:        "test",
		Endpoint:         ":8787",
		SendTimestamps:   true,
		MetricExpiration: 2 * time.Hour,
	}
	exporterFactory := NewFactory()
	set := componenttest.NewNopExporterCreateSettings()
	exporter, err := exporterFactory.CreateMetricsExporter(ctx, set, exporterCfg)
	if err != nil {
		t.Fatal(err)
	}
	if err = exporter.Start(ctx, nil); err != nil {
		t.Fatalf("Failed to start the Prometheus receiver: %v", err)
	}
	t.Cleanup(func() { require.NoError(t, exporter.Shutdown(ctx)) })

	// 3. Create the Prometheus receiver scraping from the DropWizard mock server and
	// it'll feed scraped and converted metrics then pass them to the Prometheus exporter.
	yamlConfig := []byte(fmt.Sprintf(`
        global:
          scrape_interval: 2ms
          
        scrape_configs:
            - job_name: 'otel-collector'
              scrape_interval: 2ms
              static_configs:
                - targets: ['%s']
        `, srvURL.Host))
	receiverConfig := new(promconfig.Config)
	if err = yaml.Unmarshal(yamlConfig, receiverConfig); err != nil {
		t.Fatal(err)
	}

	receiverFactory := prometheusreceiver.NewFactory()
	receiverCreateSet := componenttest.NewNopReceiverCreateSettings()
	rcvCfg := &prometheusreceiver.Config{
		PrometheusConfig: receiverConfig,
		ReceiverSettings: config.NewReceiverSettings(config.NewID("prometheus")),
	}
	// 3.5 Create the Prometheus receiver and pass in the preivously created Prometheus exporter.
	prometheusReceiver, err := receiverFactory.CreateMetricsReceiver(ctx, receiverCreateSet, rcvCfg, exporter)
	if err != nil {
		t.Fatal(err)
	}
	if err = prometheusReceiver.Start(ctx, nil); err != nil {
		t.Fatalf("Failed to start the Prometheus receiver: %v", err)
	}
	t.Cleanup(func() { require.NoError(t, prometheusReceiver.Shutdown(ctx)) })

	// 4. Scrape from the Prometheus exporter to ensure that we export summary metrics
	// We shall let the Prometheus exporter scrape the DropWizard mock server, at least 9 times.
	for i := 0; i < 8; i++ {
		<-waitForScrape
	}
	res, err := http.Get("http://localhost" + exporterCfg.Endpoint + "/metrics")
	if err != nil {
		t.Fatalf("Failed to scrape from the exporter: %v", err)
	}
	prometheusExporterScrape, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	// 5. Verify that we have the summary metrics and that their values make sense.
	wantLineRegexps := []string{
		`. HELP test_jvm_gc_collection_seconds Time spent in a given JVM garbage collector in seconds.`,
		`. TYPE test_jvm_gc_collection_seconds summary`,
		`test_jvm_gc_collection_seconds_sum.gc="G1 Old Generation". 0.*`,
		`test_jvm_gc_collection_seconds_count.gc="G1 Old Generation". 0.*`,
		`test_jvm_gc_collection_seconds_sum.gc="G1 Young Generation". 0.*`,
		`test_jvm_gc_collection_seconds_count.gc="G1 Young Generation". 9.*`,
		`. HELP test_jvm_info JVM version info`,
		`. TYPE test_jvm_info gauge`,
		`test_jvm_info.vendor="Oracle Corporation",version="9.0.4.11". 1.*`,
		`. HELP test_jvm_memory_pool_bytes_used Used bytes of a given JVM memory pool.`,
		`. TYPE test_jvm_memory_pool_bytes_used gauge`,
		`test_jvm_memory_pool_bytes_used.pool="CodeHeap 'non.nmethods'". 1.277952e.06.*`,
		`test_jvm_memory_pool_bytes_used.pool="CodeHeap 'non.profiled nmethods'". 2.869376e.06.*`,
		`test_jvm_memory_pool_bytes_used.pool="CodeHeap 'profiled nmethods'". 6.871168e.06.*`,
		`test_jvm_memory_pool_bytes_used.pool="Compressed Class Space". 2.751312e.06.*`,
		`test_jvm_memory_pool_bytes_used.pool="G1 Eden Space". 4.4040192e.07.*`,
		`test_jvm_memory_pool_bytes_used.pool="G1 Old Gen". 4.385408e.06.*`,
		`test_jvm_memory_pool_bytes_used.pool="G1 Survivor Space". 8.388608e.06.*`,
		`test_jvm_memory_pool_bytes_used.pool="Metaspace". 2.6218176e.07.*`,
		`. HELP test_scrape_duration_seconds Duration of the scrape`,
		`. TYPE test_scrape_duration_seconds gauge`,
		`test_scrape_duration_seconds [0-9.e-]+ [0-9]+`,
		`. HELP test_scrape_samples_post_metric_relabeling The number of samples remaining after metric relabeling was applied`,
		`. TYPE test_scrape_samples_post_metric_relabeling gauge`,
		`test_scrape_samples_post_metric_relabeling 13 .*`,
		`. HELP test_scrape_samples_scraped The number of samples the target exposed`,
		`. TYPE test_scrape_samples_scraped gauge`,
		`test_scrape_samples_scraped 13 .*`,
		`. HELP test_scrape_series_added The approximate number of new series in this scrape`,
		`. TYPE test_scrape_series_added gauge`,
		`test_scrape_series_added 13 .*`,
		`. HELP test_up The scraping was successful`,
		`. TYPE test_up gauge`,
		`test_up 1 .*`,
	}

	// 5.5: Perform a complete line by line prefix verification to ensure we extract back the inputs
	// we'd expect after scraping Prometheus.
	for _, wantLineRegexp := range wantLineRegexps {
		reg := regexp.MustCompile(wantLineRegexp)
		prometheusExporterScrape = reg.ReplaceAll(prometheusExporterScrape, []byte(""))
	}
	// After this replacement, there should ONLY be newlines present.
	prometheusExporterScrape = bytes.ReplaceAll(prometheusExporterScrape, []byte("\n"), []byte(""))
	// Now assert that NO output was left over.
	if len(prometheusExporterScrape) != 0 {
		t.Fatalf("Left-over unmatched Prometheus scrape content: %q\n", prometheusExporterScrape)
	}

}

const dropWizardResponse = `
# HELP jvm_memory_pool_bytes_used Used bytes of a given JVM memory pool.
# TYPE jvm_memory_pool_bytes_used gauge
jvm_memory_pool_bytes_used{pool="CodeHeap 'non-nmethods'",} 1277952.0
jvm_memory_pool_bytes_used{pool="Metaspace",} 2.6218176E7
jvm_memory_pool_bytes_used{pool="CodeHeap 'profiled nmethods'",} 6871168.0
jvm_memory_pool_bytes_used{pool="Compressed Class Space",} 2751312.0
jvm_memory_pool_bytes_used{pool="G1 Eden Space",} 4.4040192E7
jvm_memory_pool_bytes_used{pool="G1 Old Gen",} 4385408.0
jvm_memory_pool_bytes_used{pool="G1 Survivor Space",} 8388608.0
jvm_memory_pool_bytes_used{pool="CodeHeap 'non-profiled nmethods'",} 2869376.0
# HELP jvm_info JVM version info
# TYPE jvm_info gauge
jvm_info{version="9.0.4+11",vendor="Oracle Corporation",} 1.0
# HELP jvm_gc_collection_seconds Time spent in a given JVM garbage collector in seconds.
# TYPE jvm_gc_collection_seconds summary
jvm_gc_collection_seconds_count{gc="G1 Young Generation",} 9.0
jvm_gc_collection_seconds_sum{gc="G1 Young Generation",} 0.229
jvm_gc_collection_seconds_count{gc="G1 Old Generation",} 0.0
jvm_gc_collection_seconds_sum{gc="G1 Old Generation",} 0.0`
