// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stanza

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/pdata"
)

const (
	DefaultFlushInterval = 100 * time.Millisecond
	DefaultMaxFlushCount = 100
)

// Converter converts entry.Entry into pdata.Logs aggregating translated
// entries into logs coming from the same Resource.
// Logs are being sent out based on the flush interval and/or the maximum
// batch size.
//
// The diagram below illustrates the internal communication inside the Converter:
//
//            ┌─────────────────────────────────┐
//            │ Batch()                         │
//  ┌─────────┤  Ingests log entries and sends  │
//  │         │  them onto a workerChan         │
//  │         └─────────────────────────────────┘
//  │
//  │ ┌───────────────────────────────────────────────────┐
//  ├─► workerLoop()                                      │
//  │ │ ┌─────────────────────────────────────────────────┴─┐
//  ├─┼─► workerLoop()                                      │
//  │ │ │ ┌─────────────────────────────────────────────────┴─┐
//  └─┼─┼─► workerLoop()                                      │
//    └─┤ │   consumes sent log entries from workerChan,      │
//      │ │   translates received entries to pdata.LogRecords,│
//      └─┤   marshalls them to JSON and send them onto       │
//        │   batchChan                                       │
//        └─────────────────────────┬─────────────────────────┘
//                                  │
//                                  ▼
//      ┌─────────────────────────────────────────────────────┐
//      │ batchLoop()                                         │
//      │   consumes from batchChan, aggregates log records   │
//      │   by marshaled Resource and based on flush interval │
//      │   and maxFlushCount decides whether to send the     │
//      │   aggregated buffer to flushChan                    │
//      └───────────────────────────┬─────────────────────────┘
//                                  │
//                                  ▼
//      ┌─────────────────────────────────────────────────────┐
//      │ flushLoop()                                         │
//      │   receives log records from flushChan and sends     │
//      │   them onto pLogsChan which is consumed by          │
//      │   downstream consumers via OutChannel()             │
//      └─────────────────────────────────────────────────────┘
//
type Converter struct {
	// pLogsChan is a channel on which batched logs will be sent to.
	pLogsChan chan pdata.Logs

	stopOnce sync.Once
	stopChan chan struct{}

	// workerChan is an internal communication channel that gets the log
	// entries from Batch() calls and it receives the data in workerLoop().
	workerChan chan *entry.Entry
	// workerCount configures the amount of workers started.
	workerCount int
	// batchChan obtains log entries converted by the pool of workers,
	// in a form of logRecords grouped by Resource and then after aggregating
	// them decides based on maxFlushCount if the flush should be triggered.
	// If also serves the ticker flushes configured by flushInterval.
	batchChan chan *workerItem

	// flushInterval defines how often we flush the aggregated log entries.
	flushInterval time.Duration
	// maxFlushCount defines what's the amount of entries in the buffer that
	// will trigger a flush of log entries.
	maxFlushCount uint
	// flushChan is an internal channel used for transporting batched pdata.Logs.
	flushChan chan pdata.Logs

	// data holds currently converted and aggregated log entries, grouped by Resource.
	data map[string]pdata.Logs
	// logRecordCount holds the number of translated and accumulated log Records
	// and is compared against maxFlushCount to make a decision whether to flush.
	logRecordCount uint

	// wg is a WaitGroup that makes sure that we wait for spun up goroutines exit
	// when Stop() is called.
	wg sync.WaitGroup

	logger *zap.Logger
}

type ConverterOption interface {
	apply(*Converter)
}

type optionFunc func(*Converter)

func (f optionFunc) apply(c *Converter) {
	f(c)
}

func WithFlushInterval(interval time.Duration) ConverterOption {
	return optionFunc(func(c *Converter) {
		c.flushInterval = interval
	})
}

func WithMaxFlushCount(count uint) ConverterOption {
	return optionFunc(func(c *Converter) {
		c.maxFlushCount = count
	})
}

func WithLogger(logger *zap.Logger) ConverterOption {
	return optionFunc(func(c *Converter) {
		c.logger = logger
	})
}

func WithWorkerCount(workerCount int) ConverterOption {
	return optionFunc(func(c *Converter) {
		c.workerCount = workerCount
	})
}

func NewConverter(opts ...ConverterOption) *Converter {
	c := &Converter{
		workerChan:    make(chan *entry.Entry),
		workerCount:   runtime.NumCPU() / 4,
		batchChan:     make(chan *workerItem),
		data:          make(map[string]pdata.Logs),
		pLogsChan:     make(chan pdata.Logs),
		stopChan:      make(chan struct{}),
		logger:        zap.NewNop(),
		flushChan:     make(chan pdata.Logs),
		flushInterval: DefaultFlushInterval,
		maxFlushCount: DefaultMaxFlushCount,
	}

	for _, opt := range opts {
		opt.apply(c)
	}

	return c
}

func (c *Converter) Start() {
	c.logger.Debug("Starting log converter", zap.Int("worker_count", c.workerCount))

	for i := 0; i < c.workerCount; i++ {
		c.wg.Add(1)
		go c.workerLoop()
	}

	c.wg.Add(1)
	go c.batchLoop()

	c.wg.Add(1)
	go c.flushLoop()
}

func (c *Converter) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopChan)
		c.wg.Wait()
		close(c.pLogsChan)
	})
}

// Channel returns the channel on which converted entries will be sent to.
func (c *Converter) OutChannel() <-chan pdata.Logs {
	return c.pLogsChan
}

type workerItem struct {
	Resource       map[string]string
	LogRecord      pdata.LogRecord
	ResourceString string
}

// workerLoop is responsible for obtaining log entries from Batch() calls,
// converting them to pdata.LogRecords and sending them together with the
// associated Resource through the batchChan for aggregation.
func (c *Converter) workerLoop() {
	defer c.wg.Done()

	var (
		buff    = bytes.Buffer{}
		encoder = json.NewEncoder(&buff)
	)

	for {

		select {
		case <-c.stopChan:
			return

		case e, ok := <-c.workerChan:
			if !ok {
				return
			}

			buff.Reset()
			lr := convert(e)

			if err := encoder.Encode(e.Resource); err != nil {
				c.logger.Debug("Failed marshaling entry.Resource to JSON",
					zap.Any("resource", e.Resource),
				)
				continue
			}

			select {
			case c.batchChan <- &workerItem{
				Resource:       e.Resource,
				ResourceString: buff.String(),
				LogRecord:      lr,
			}:
			case <-c.stopChan:
			}
		}
	}
}

// batchLoop is responsible for receiving the converted log entries and aggregating
// them by Resource.
// Whenever maxFlushCount is reached or the ticker ticks a flush is triggered.
func (c *Converter) batchLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case wi, ok := <-c.batchChan:
			if !ok {
				return
			}

			pLogs, ok := c.data[wi.ResourceString]
			if ok {
				pLogs.ResourceLogs().
					At(0).InstrumentationLibraryLogs().
					At(0).Logs().Append(wi.LogRecord)
			} else {
				pLogs = pdata.NewLogs()
				logs := pLogs.ResourceLogs()
				logs.Resize(1)
				rls := logs.At(0)

				resource := rls.Resource()
				resourceAtts := resource.Attributes()
				resourceAtts.EnsureCapacity(len(wi.Resource))
				for k, v := range wi.Resource {
					resourceAtts.InsertString(k, v)
				}

				ills := rls.InstrumentationLibraryLogs()
				ills.Resize(1)
				ills.At(0).Logs().Append(wi.LogRecord)
			}

			c.data[wi.ResourceString] = pLogs
			c.logRecordCount++

			if c.logRecordCount >= c.maxFlushCount {
				for r, pLogs := range c.data {
					c.flushChan <- pLogs
					delete(c.data, r)
				}
				c.logRecordCount = 0
			}

		case <-ticker.C:
			for r, pLogs := range c.data {
				c.flushChan <- pLogs
				delete(c.data, r)
			}
			c.logRecordCount = 0

		case <-c.stopChan:
			return
		}
	}
}

func (c *Converter) flushLoop() {
	defer c.wg.Done()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		select {
		case <-c.stopChan:
			return

		case pLogs := <-c.flushChan:
			if err := c.flush(ctx, pLogs); err != nil {
				c.logger.Debug("Problem sending log entries",
					zap.Error(err),
				)
			}
		}
	}
}

// flush flushes provided pdata.Logs entries onto a channel.
func (c *Converter) flush(ctx context.Context, pLogs pdata.Logs) error {
	doneChan := ctx.Done()

	select {
	case <-doneChan:
		return fmt.Errorf("flushing log entries interrupted, err: %w", ctx.Err())

	case c.pLogsChan <- pLogs:

	// The converter has been stopped so bail the flush.
	case <-c.stopChan:
		return errors.New("Logs converter has been stopped")
	}

	return nil
}

// Batch takes in an entry.Entry and sends it to an available worker for processing.
func (c *Converter) Batch(e *entry.Entry) error {
	select {
	case c.workerChan <- e:
		return nil
	case <-c.stopChan:
		return errors.New("Logs converter has been stopped")
	}
}

// convert converts one entry.Entry into pdata.LogRecord allocating it.
func convert(ent *entry.Entry) pdata.LogRecord {
	dest := pdata.NewLogRecord()
	convertInto(ent, dest)
	return dest
}

// convertInto converts entry.Entry into provided pdata.LogRecord.
func convertInto(ent *entry.Entry, dest pdata.LogRecord) {
	dest.SetTimestamp(pdata.TimestampFromTime(ent.Timestamp))

	sevText, sevNum := convertSeverity(ent.Severity)
	dest.SetSeverityText(sevText)
	dest.SetSeverityNumber(sevNum)

	if l := len(ent.Attributes); l > 0 {
		attributes := dest.Attributes()
		attributes.EnsureCapacity(l)
		for k, v := range ent.Attributes {
			attributes.InsertString(k, v)
		}
	}

	insertToAttributeVal(ent.Body, dest.Body())

	if ent.TraceId != nil {
		var buffer [16]byte
		copy(buffer[0:16], ent.TraceId)
		dest.SetTraceID(pdata.NewTraceID(buffer))
	}
	if ent.SpanId != nil {
		var buffer [8]byte
		copy(buffer[0:8], ent.SpanId)
		dest.SetSpanID(pdata.NewSpanID(buffer))
	}
	if ent.TraceFlags != nil {
		// The 8 least significant bits are the trace flags as defined in W3C Trace
		// Context specification. Don't override the 24 reserved bits.
		flags := dest.Flags()
		flags &= 0xFFFFFF00
		flags |= uint32(ent.TraceFlags[0])
		dest.SetFlags(flags)
	}
}

func insertToAttributeVal(value interface{}, dest pdata.AttributeValue) {
	switch t := value.(type) {
	case bool:
		dest.SetBoolVal(t)
	case string:
		dest.SetStringVal(t)
	case []byte:
		dest.SetStringVal(string(t))
	case int64:
		dest.SetIntVal(t)
	case int32:
		dest.SetIntVal(int64(t))
	case int16:
		dest.SetIntVal(int64(t))
	case int8:
		dest.SetIntVal(int64(t))
	case int:
		dest.SetIntVal(int64(t))
	case uint64:
		dest.SetIntVal(int64(t))
	case uint32:
		dest.SetIntVal(int64(t))
	case uint16:
		dest.SetIntVal(int64(t))
	case uint8:
		dest.SetIntVal(int64(t))
	case uint:
		dest.SetIntVal(int64(t))
	case float64:
		dest.SetDoubleVal(t)
	case float32:
		dest.SetDoubleVal(float64(t))
	case map[string]interface{}:
		toAttributeMap(t).CopyTo(dest)
	case []interface{}:
		toAttributeArray(t).CopyTo(dest)
	default:
		dest.SetStringVal(fmt.Sprintf("%v", t))
	}
}

func toAttributeMap(obsMap map[string]interface{}) pdata.AttributeValue {
	attVal := pdata.NewAttributeValueMap()
	attMap := attVal.MapVal()
	attMap.EnsureCapacity(len(obsMap))
	for k, v := range obsMap {
		switch t := v.(type) {
		case bool:
			attMap.InsertBool(k, t)
		case string:
			attMap.InsertString(k, t)
		case []byte:
			attMap.InsertString(k, string(t))
		case int64:
			attMap.InsertInt(k, t)
		case int32:
			attMap.InsertInt(k, int64(t))
		case int16:
			attMap.InsertInt(k, int64(t))
		case int8:
			attMap.InsertInt(k, int64(t))
		case int:
			attMap.InsertInt(k, int64(t))
		case uint64:
			attMap.InsertInt(k, int64(t))
		case uint32:
			attMap.InsertInt(k, int64(t))
		case uint16:
			attMap.InsertInt(k, int64(t))
		case uint8:
			attMap.InsertInt(k, int64(t))
		case uint:
			attMap.InsertInt(k, int64(t))
		case float64:
			attMap.InsertDouble(k, t)
		case float32:
			attMap.InsertDouble(k, float64(t))
		case map[string]interface{}:
			subMap := toAttributeMap(t)
			attMap.Insert(k, subMap)
		case []interface{}:
			arr := toAttributeArray(t)
			attMap.Insert(k, arr)
		default:
			attMap.InsertString(k, fmt.Sprintf("%v", t))
		}
	}
	return attVal
}

func toAttributeArray(obsArr []interface{}) pdata.AttributeValue {
	arrVal := pdata.NewAttributeValueArray()
	arr := arrVal.ArrayVal()
	arr.Resize(len(obsArr))
	for i, v := range obsArr {
		insertToAttributeVal(v, arr.At(i))
	}
	return arrVal
}

func convertSeverity(s entry.Severity) (string, pdata.SeverityNumber) {
	switch {

	// Handle standard severity levels
	case s == entry.Catastrophe:
		return "Fatal", pdata.SeverityNumberFATAL4
	case s == entry.Emergency:
		return "Error", pdata.SeverityNumberFATAL
	case s == entry.Alert:
		return "Error", pdata.SeverityNumberERROR3
	case s == entry.Critical:
		return "Error", pdata.SeverityNumberERROR2
	case s == entry.Error:
		return "Error", pdata.SeverityNumberERROR
	case s == entry.Warning:
		return "Info", pdata.SeverityNumberINFO4
	case s == entry.Notice:
		return "Info", pdata.SeverityNumberINFO3
	case s == entry.Info:
		return "Info", pdata.SeverityNumberINFO
	case s == entry.Debug:
		return "Debug", pdata.SeverityNumberDEBUG
	case s == entry.Trace:
		return "Trace", pdata.SeverityNumberTRACE2

	// Handle custom severity levels
	case s > entry.Emergency:
		return "Fatal", pdata.SeverityNumberFATAL2
	case s > entry.Alert:
		return "Error", pdata.SeverityNumberERROR4
	case s > entry.Critical:
		return "Error", pdata.SeverityNumberERROR3
	case s > entry.Error:
		return "Error", pdata.SeverityNumberERROR2
	case s > entry.Warning:
		return "Info", pdata.SeverityNumberINFO4
	case s > entry.Notice:
		return "Info", pdata.SeverityNumberINFO3
	case s > entry.Info:
		return "Info", pdata.SeverityNumberINFO2
	case s > entry.Debug:
		return "Debug", pdata.SeverityNumberDEBUG2
	case s > entry.Trace:
		return "Trace", pdata.SeverityNumberTRACE3
	case s > entry.Default:
		return "Trace", pdata.SeverityNumberTRACE

	default:
		return "Undefined", pdata.SeverityNumberUNDEFINED
	}
}
