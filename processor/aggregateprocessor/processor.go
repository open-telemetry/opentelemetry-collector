// Copyright 2019, OpenTelemetry Authors
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

package aggregateprocessor

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"go.opencensus.io/stats"
	"go.uber.org/zap"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/facette/natsort"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/exporter/opencensusexporter"
	"go.opentelemetry.io/collector/translator/internaldata"
)

// aggregatingProcessor is the component that fowards spans to collector peers
// based on traceID
type aggregatingProcessor struct {
	//context
	ctx context.Context
	// to start the memberSyncTicker
	start sync.Once
	// member lock
	lock sync.RWMutex
	// nextConsumer
	nextConsumer consumer.TraceConsumer
	// logger
	logger *zap.Logger
	// self member IP
	selfIP string
	// ringMemberQuerier instance
	ring ringMemberQuerier
	// peer port
	peerPort int
	// ticker to call ring.GetState() and sync member list
	memberSyncTicker tTicker
	// exporters for each of the collector peers
	collectorPeers map[string]component.TraceExporter
}

type testProcessor struct {
	nextConsumer consumer.TraceConsumer
}

func newTraceProcessor(logger *zap.Logger, nextConsumer consumer.TraceConsumer, cfg Config) (component.TraceProcessor, error) {
	if nextConsumer == nil {
		return nil, componenterror.ErrNilNextConsumer
	}

	if cfg.PeerDiscoveryDNSName == "" {
		return nil, fmt.Errorf("peer discovery DNS name not provided")
	}

	if cfg.PeerPort == 0 {
		return nil, fmt.Errorf("nil peer port")
	}

	ap := &aggregatingProcessor{
		ctx: context.Background(),
		nextConsumer:   nextConsumer,
		logger:         logger,
		collectorPeers: make(map[string]component.TraceExporter),
		peerPort:       cfg.PeerPort,
	}

	ap.ring = newRingMemberQuerier(logger, cfg.PeerDiscoveryDNSName)

	if ip, err := externalIP(); err == nil {
		ap.selfIP = ip
	} else {
		return nil, err
	}

	ap.memberSyncTicker = &policyTicker{onTick: ap.memberSyncOnTick}
	return ap, nil
}


func newTraceExporter(logger *zap.Logger, ip string, peerPort int) component.TraceExporter {
	factory := opencensusexporter.NewFactory()
	config := factory.CreateDefaultConfig()
	config.(*opencensusexporter.Config).ExporterSettings = configmodels.ExporterSettings{
		NameVal: "opencensus",
		TypeVal: "opencensus",
	}
	config.(*opencensusexporter.Config).GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: ip + ":" + strconv.Itoa(peerPort),
	}

	logger.Info("Creating new exporter for", zap.String("PeerIP", ip))
	// TODO: revisit the context.Background() here
	exporter, err := factory.CreateTraceExporter(context.Background(), component.ExporterCreateParams{Logger: logger}, config)
	if err != nil {
		logger.Fatal("Could not create span exporter", zap.Error(err))
		return nil
	}

	return exporter
}

// At the set frequency, get the state of the collector peer list
func (ap *aggregatingProcessor) memberSyncOnTick() {
	newMembers := ap.ring.getMembers()
	if newMembers == nil {
		return
	}

	// check if member list has changed
	ap.lock.RLock()
	curMembers := make([]string, 0, len(ap.collectorPeers))
	for k := range ap.collectorPeers {
		curMembers = append(curMembers, k)
	}
	ap.lock.RUnlock()

	natsort.Sort(curMembers)
	natsort.Sort(newMembers)

	// checking if curMembers == newMembers
	isEqual := true
	if len(curMembers) != len(newMembers) {
		isEqual = false
	} else {
		for k, v := range curMembers {
			if v != newMembers[k] {
				isEqual = false
			}
		}
	}

	if !isEqual {
		// Remove old members
		// Find diff(curMembers, newMembers)
		for _, c := range curMembers {
			// check if c is part of newMembers
			flag := 0
			for _, n := range newMembers {
				if c == n {
					flag = 1
				}
			}
			if flag == 0 {
				// Need a write lock here
				ap.lock.Lock()
				// nullify the collector peer instance
				ap.collectorPeers[c] = nil
				// delete the key
				delete(ap.collectorPeers, c)
				ap.lock.Unlock()
				ap.logger.Info("(memberSyncOnTick) Deleted member", zap.String("Member ip", c))
			}
		}
		// Add new members
		for _, v := range newMembers {
			if _, ok := ap.collectorPeers[v]; ok {
				// exists, do nothing
			} else if v == ap.selfIP {
				// Need a write lock here
				ap.lock.Lock()
				ap.collectorPeers[v] = nil
				ap.lock.Unlock()
			} else {
				newPeer := newTraceExporter(ap.logger, v, ap.peerPort)
				if newPeer == nil {
					return
				}
				// Need a write lock here
				ap.lock.Lock()
				// build a new trace exporter
				ap.collectorPeers[v] = newPeer
				ap.lock.Unlock()
				ap.logger.Info("(memberSyncOnTick) Added member", zap.String("Member ip", v))
			}
		}
	}
}

func (ap *aggregatingProcessor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	octds := internaldata.TraceDataToOC(td)
	var errs []error
	for _, octd := range octds {
		if err := ap.processTraces(ctx, octd); err != nil {
			errs = append(errs, err)
		}
	}
	return componenterror.CombineErrors(errs)
}

func(ap *aggregatingProcessor) processTraces(ctx context.Context, td consumerdata.TraceData) error {
	ap.start.Do(func() {
		ap.logger.Info("First span received, starting member sync timer")
		// Run first one manually
		ap.memberSyncOnTick()
		ap.memberSyncTicker.Start(10000 * time.Millisecond) // 10s
	})

	if len(td.Spans) == 0 {
		return fmt.Errorf("empty batch")
	}

	stats.Record(ctx, statCountSpansReceived.M(int64(len(td.Spans))))

	ap.lock.RLock()
	defer ap.lock.RUnlock()

	// get members:
	peersSorted := []string{}
	for k := range ap.collectorPeers {
		peersSorted = append(peersSorted, k)
	}
	natsort.Sort(peersSorted)
	ap.logger.Debug("Printing member list", zap.Strings("Members", peersSorted))

	// build batches for every member
	var batches [][]*tracepb.Span
	for p := 0; p < len(peersSorted); p++ {
		batches = append(batches, make([]*tracepb.Span, 0))
	}

	// Should ideally be empty
	noHashBatch := []*tracepb.Span{}

	for _, span := range td.Spans {
		memberNum := jumpHash(fingerprint(span.TraceId), len(peersSorted))
		ap.logger.Debug("", zap.Int("memberNum", int(memberNum)))
		if memberNum == -1 {
			// Any spans having a hash error -> self processed
			noHashBatch = append(noHashBatch, span)
		} else {
			// Append this span to the batch of that member
			batches[memberNum] = append(batches[memberNum], span)
		}
	}

	for k, batch := range batches {
		curIP := peersSorted[k]
		if curIP == ap.selfIP {
			batch = append(batch, noHashBatch...)
			toSend := consumerdata.TraceData{Spans: batch}
			ap.nextConsumer.ConsumeTraces(ctx, internaldata.OCToTraceData(toSend))
			continue
		} else {
			// track metrics for forwarded spans
			stats.Record(ctx, statCountSpansForwarded.M(int64(len(batch))))
		}
		ap.logger.Debug("Sending batch to collector peer", zap.String("PeerIP", curIP), zap.Int("Batch-size", len(batch)))
		toSend := consumerdata.TraceData{Spans: batch}
		ap.nextConsumer.ConsumeTraces(ctx, internaldata.OCToTraceData(toSend))
	}

	return nil
}

func (ap *aggregatingProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: false}
}

// Start is invoked during service startup.
func (ap *aggregatingProcessor) Start(context.Context, component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (ap *aggregatingProcessor) Shutdown(ctx context.Context) error {
	ap.memberSyncTicker.Stop()
	return nil
}

// tTicker interface allows easier testing of ticker related functionality used by tailSamplingProcessor
type tTicker interface {
	// Start sets the frequency of the ticker and starts the periodic calls to OnTick.
	Start(d time.Duration)
	// OnTick is called when the ticker fires.
	OnTick()
	// Stops firing the ticker.
	Stop()
}

type policyTicker struct {
	ticker *time.Ticker
	onTick func()
}

func (pt *policyTicker) Start(d time.Duration) {
	pt.ticker = time.NewTicker(d)
	go func() {
		for range pt.ticker.C {
			pt.OnTick()
		}
	}()
}
func (pt *policyTicker) OnTick() {
	pt.onTick()
}
func (pt *policyTicker) Stop() {
	pt.ticker.Stop()
}

var _ tTicker = (*policyTicker)(nil)
