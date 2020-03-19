package receivertest

import (
	"context"
	"sync"

	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
)

type TraceAppender struct {
	sync.RWMutex
	TraceDataList []consumerdata.TraceData
}

func NewTraceAppender() *TraceAppender {
	return &TraceAppender{}
}

var _ consumer.TraceConsumer = (*TraceAppender)(nil)

func (ma *TraceAppender) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	ma.Lock()
	defer ma.Unlock()

	ma.TraceDataList = append(ma.TraceDataList, td)

	return nil
}
