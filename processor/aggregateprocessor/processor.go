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
	"errors"
	"fmt"
	"hash/fnv"
	"net"
	"strconv"
	"sync"
	"time"

	"go.opencensus.io/stats"
	"go.uber.org/zap"

	v1 "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/facette/natsort"
	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configgrpc"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter/opencensusexporter"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
)

// aggregatingProcessor is the component that fowards spans to collector peers
// based on traceID
type aggregatingProcessor struct {
	// to start the memberSyncTicker
	start sync.Once
	// member lock
	lock sync.RWMutex
	// nextConsumer
	nextConsumer consumer.TraceConsumerOld
	// logger
	logger *zap.Logger
	// peerDiscoveryName
	peerDiscoveryName string
	// self member IP
	selfIP string
	// ticker to call ring.GetState() and sync member list
	memberSyncTicker tTicker
	// exporters for each of the collector peers
	collectorPeers map[string]component.TraceExporterOld
}

var _ component.TraceProcessorOld = (*aggregatingProcessor)(nil)

func newTraceProcessor(logger *zap.Logger, nextConsumer consumer.TraceConsumerOld, cfg *Config) (component.TraceProcessorOld, error) {
	if nextConsumer == nil {
		return nil, oterr.ErrNilNextConsumer
	}

	if cfg.PeerDiscoveryDNSName != "" {
		return nil, fmt.Errorf("peer discovery DNS name not provided")
	}

	ap := &aggregatingProcessor{
		nextConsumer:      nextConsumer,
		logger:            logger,
		peerDiscoveryName: cfg.PeerDiscoveryDNSName,
		collectorPeers:    make(map[string]component.TraceExporterOld),
	}

	if ip, err := externalIP(); err == nil {
		ap.selfIP = ip
	} else {
		return nil, err
	}

	ap.memberSyncTicker = &policyTicker{onTick: ap.memberSyncOnTick}

	return ap, nil
}

func newTraceExporter(logger *zap.Logger, ip string) component.TraceExporterOld {
	factory := &opencensusexporter.Factory{}
	config := factory.CreateDefaultConfig()
	config.(*opencensusexporter.Config).ExporterSettings = configmodels.ExporterSettings{
		NameVal: "opencensus",
		TypeVal: "opencensus",
	}
	config.(*opencensusexporter.Config).GRPCSettings = configgrpc.GRPCSettings{
		Endpoint: ip + ":" + strconv.Itoa(55678),
	}

	logger.Info("Creating new exporter for", zap.String("PeerIP", ip))
	exporter, err := factory.CreateTraceExporter(logger, config)
	if err != nil {
		logger.Fatal("Could not create span exporter", zap.Error(err))
		return nil
	}

	return exporter
}

// At the set frequency, get the state of the collector peer list
func (ap *aggregatingProcessor) memberSyncOnTick() {
	ap.logger.Debug("Pooling a dns endpoint")
	// poll a dns endpoint
	ips, err := net.LookupIP(ap.peerDiscoveryName)
	if err != nil {
		ap.logger.Info("DNS lookup error", zap.Error(err))
		return
	}
	// check if this list has diverged from the current list
	var newMembers []string
	for _, v := range ips {
		newMembers = append(newMembers, v.String())
	}

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
				newPeer := newTraceExporter(ap.logger, v)
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

func externalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

// Copied from github.com/dgryski/go-jump/blob/master/jump.go
func jumpHash(key uint64, numBuckets int) int32 {

	var b int64 = -1
	var j int64

	for j < int64(numBuckets) {
		b = j
		key = key*2862933555777941757 + 1
		j = int64(float64(b+1) * (float64(int64(1)<<31) / float64((key>>33)+1)))
	}

	return int32(b)
}

func fingerprint(b []byte) uint64 {
	hash := fnv.New64a()
	hash.Write(b)
	return hash.Sum64()
}

func (ap *aggregatingProcessor) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	// Start member sync
	ap.start.Do(func() {
		ap.logger.Info("First span received, starting member sync timer")
		// Run first one manually
		ap.memberSyncOnTick()
		ap.memberSyncTicker.Start(10000 * time.Millisecond) // 10s
	})

	if td.Spans == nil {
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
	var batches [][]*v1.Span
	for p := 0; p < len(peersSorted); p++ {
		batches = append(batches, make([]*v1.Span, 0))
	}

	// Should ideally be empty
	noHashBatch := []*v1.Span{}

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
			ap.nextConsumer.ConsumeTraceData(ctx, toSend)
			continue
		} else {
			// track metrics for forwarded spans
			stats.Record(ctx, statCountSpansForwarded.M(int64(len(batch))))
		}
		ap.logger.Debug("Sending batch to collector peer", zap.String("PeerIP", curIP), zap.Int("Batch-size", len(batch)))
		toSend := consumerdata.TraceData{Spans: batch}
		ap.collectorPeers[curIP].ConsumeTraceData(ctx, toSend)
	}

	return nil
}

func (ap *aggregatingProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: false}
}

// Start is invoked during service startup.
func (ap *aggregatingProcessor) Start(host component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (ap *aggregatingProcessor) Shutdown() error {
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
