package receivertest

import (
	"context"
	"sync"

	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
)

type MetricAppender struct {
	sync.RWMutex
	MetricsDataList []consumerdata.MetricsData
}

func NewMetricAppender() *MetricAppender {
	return &MetricAppender{}
}

var _ consumer.MetricsConsumer = (*MetricAppender)(nil)

func (ma *MetricAppender) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	ma.Lock()
	defer ma.Unlock()

	ma.MetricsDataList = append(ma.MetricsDataList, md)

	return nil
}
