package summaryapireceiver

import (
	"context"
	"errors"

	// "encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	// "k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
	"net/http"

	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-service/consumer"
)

// SummaryApiCollector is a struct that collects and reports metrics from the Kubernetes kubelet's Summary API
type SummaryApiCollector struct {
	consumer        consumer.MetricsConsumer
	startTime       time.Time
	kubeletEndpoint string
	scrapeInterval  time.Duration
	metricPrefix    string
	done            chan struct{}
}

const defaultScrapeInterval   = 60 * time.Second
var errEmptyKubeletEndpoint = errors.New("expecting a non-empty kubelet_endpoint")

// NewSummaryApiCollector creates a new collector for the kubelet's Summary API
func NewSummaryApiCollector(si time.Duration, kubeletEndpoint, prefix string, consumer consumer.MetricsConsumer) (*SummaryApiCollector, error) {
	if kubeletEndpoint == "" {
		return nil, errEmptyKubeletEndpoint
	}
	if si <= 0 {
		si = defaultScrapeInterval
	}
	sac := &SummaryApiCollector{
		consumer:        consumer,
		startTime:       time.Now(),
		kubeletEndpoint: kubeletEndpoint,
		scrapeInterval:  si,
		metricPrefix:    prefix,
		done:            make(chan struct{}),
	}

	return sac, nil
}

// StartCollection starts a ticker'd goroutine that will scrape and export vm metrics periodically.
func (sac *SummaryApiCollector) StartCollection() {
	go func() {
		ticker := time.NewTicker(sac.scrapeInterval)
		for {
			select {
			case <-ticker.C:
				sac.scrapeAndExport()

			case <-sac.done:
				return
			}
		}
	}()
}

// StopCollection stops the collection of metric information
func (sac *SummaryApiCollector) StopCollection() {
	close(sac.done)
}

func (sac *SummaryApiCollector) scrapeAndExport() {
	_, span := trace.StartSpan(context.Background(), "SummaryApiCollector.scrapeAndExport")
	defer span.End()

	// metrics := make([]*metricspb.Metric, 0, len(vmMetricDescriptors))
	resp, err := http.Get(sac.kubeletEndpoint)
	if resp != nil {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		str := string(body)
		// TODO(dpo): uncomment these out once dependency on Kubernete's Summary has been resolved
		//res := Summary{}
		//json.Unmarshal([]byte(str), &res)
		//fmt.Println("resp:", res)
		fmt.Println("resp:", str)
	} else {
		fmt.Println("resp err:", err)
	}
}
