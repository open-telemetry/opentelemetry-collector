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

package tailsamplingprocessor

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"net"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	v1 "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter"
	"github.com/open-telemetry/opentelemetry-collector/exporter/opencensusexporter"
	"github.com/open-telemetry/opentelemetry-collector/extension"
	"github.com/open-telemetry/opentelemetry-collector/processor/samplingprocessor/tailsamplingprocessor/idbatcher"
)

type forwarder interface {
	// process span
	process(span *tracepb.Span) bool
	// add support extensions
	addRingMembershipExtension(ext extension.SupportExtension)
}

type collectorPeer struct {
	exporter           exporter.TraceExporter
	peerBatcher        idbatcher.Batcher
	spanDispatchTicker tTicker // to pop from the batcher and forward to peer
	logger             *zap.Logger
	start              sync.Once
	idToSpans          sync.Map
}

// spanForwarder is the component that fowards spans to collector peers
// based on traceID
type spanForwarder struct {
	// to make peerQueues concurrently accessible
	sync.RWMutex
	// to start the memberSyncTicker
	start sync.Once
	// logger
	logger *zap.Logger
	// self member IP
	selfMemberIP string
	// ticker to call ring.GetState() and sync member list
	memberSyncTicker tTicker
	// stores queues for each of the collector peers
	peerQueues map[string]*collectorPeer
	// The ringmembership extension (implements extension.SupportExtension)
	// Ownership should be with the component that requires it.
	// However it will be Start()'ed by the Application service.
	ring extension.SupportExtension
}

func (c *collectorPeer) batchDispatchOnTick() {
	c.logger.Info("Collector peer batchDispatchOnTick invoked")
	batchIds, _ := c.peerBatcher.CloseCurrentAndTakeFirstBatch()
	// create batch from batchIds
	var td consumerdata.TraceData
	for _, v := range batchIds {
		span, _ := c.idToSpans.Load(string(v))
		if span != nil {
			td.Spans = append(td.Spans, span.(*v1.Span))
		}
	}
	// simply post this batch via grpc to the collector peer
	c.logger.Info("Sending batch to collector peer")
	c.exporter.ConsumeTraceData(context.Background(), td)
}

func newCollectorPeer(logger *zap.Logger, ip string) *collectorPeer {
	factory := &opencensusexporter.Factory{}
	config := factory.CreateDefaultConfig()
	config.(*opencensusexporter.Config).GRPCSettings.Endpoint = ip + ":" + strconv.Itoa(55678)

	exporter, err := opencensusexporter.NewTraceExporter(logger, config)
	if err != nil {
		logger.Fatal("Could not create span exporter", zap.Error(err))
		return nil
	}

	batcher, err := idbatcher.New(10, 64, uint64(runtime.NumCPU()))
	if err != nil {
		logger.Fatal("Could not create id batcher", zap.Error(err))
		return nil
	}

	cp := &collectorPeer{
		exporter:    exporter,
		logger:      logger,
		peerBatcher: batcher,
	}
	cp.spanDispatchTicker = &policyTicker{onTick: cp.batchDispatchOnTick}
	return cp
}

// At the set frequency, get the state of the collector peer list
func (sf *spanForwarder) memberSyncOnTick() {
	// Get sorted member list from the extension
	state, err := sf.ring.GetState()
	if err != nil {
		return
	}

	newMembers := state.([]string)
	curMembers := make([]string, 0, len(sf.peerQueues))

	sf.RLock()
	for k := range sf.peerQueues {
		curMembers = append(curMembers, k)
	}
	sf.RUnlock()

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
			// check if v is part of newMembers
			flag := 0
			for _, n := range newMembers {
				if c == n {
					flag = 1
				}
			}

			if flag == 0 {
				// Need a write lock here
				sf.Lock()
				// nullify the collector peer instance
				sf.peerQueues[c] = nil
				// delete the key
				delete(sf.peerQueues, c)
				sf.Unlock()
			}
		}

		// Add new members
		for _, v := range newMembers {
			if _, ok := sf.peerQueues[v]; ok {
				// exists, do nothing
			} else {
				newPeer := newCollectorPeer(sf.logger, v)
				if newPeer == nil {
					return
				}
				// Need a write lock here
				sf.Lock()
				// build a new collectorPeer object
				sf.peerQueues[v] = newPeer
				sf.Unlock()
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

func newSpanForwarder(logger *zap.Logger) (forwarder, error) {
	sf := &spanForwarder{
		logger:       logger,
		selfMemberIP: "0.0.0.0",
	}

	if ip, err := externalIP(); err == nil {
		sf.selfMemberIP = ip
	}

	sf.peerQueues = make(map[string]*collectorPeer)
	sf.memberSyncTicker = &policyTicker{onTick: sf.memberSyncOnTick}

	return sf, nil
}

func getHash(traceID []byte) int64 {
	var n int64
	// simplistic for now
	buf := bytes.NewBuffer(traceID)
	binary.Read(buf, binary.LittleEndian, &n)
	return n
}

func bytesToInt(spanID []byte) int64 {
	var n int64
	// simplistic for now
	buf := bytes.NewBuffer(spanID)
	binary.Read(buf, binary.LittleEndian, &n)
	return n
}

// TODO: Use batch here instead of span
func (sf *spanForwarder) process(span *tracepb.Span) bool {
	// Start member sync
	sf.start.Do(func() {
		sf.logger.Info("First span received, starting member sync timer")
		// Run first one manually
		sf.memberSyncOnTick()
		sf.memberSyncTicker.Start(100 * time.Millisecond)
	})

	// check hash of traceid
	traceIDHash := getHash(span.TraceId)

	// The only time we need to acquire the lock is to see peer list
	sf.RLock()
	memberNum := traceIDHash % int64(len(sf.peerQueues))
	memberID := reflect.ValueOf(sf.peerQueues).MapKeys()[memberNum].Interface().(string)
	sf.RUnlock()

	if memberID == sf.selfMemberIP {
		// span should be processed by this collector peer
		return false
	}

	// Append this span to the batch of that member
	peer := sf.peerQueues[memberID]

	peer.start.Do(func() {
		peer.logger.Info("First span received, starting collector peer timers")
		peer.spanDispatchTicker.Start(100 * time.Millisecond)
	})

	// there might be multiple spans of the same traceId
	// we don't want to build the trace here(?)
	peer.peerBatcher.AddToCurrentBatch(span.SpanId)

	// Store the span in idToSpans
	peer.idToSpans.Store(string(span.SpanId), span)

	return true
}

func (sf *spanForwarder) addRingMembershipExtension(ext extension.SupportExtension) {
	sf.logger.Info("Added ring membership extension to span forwarder in tail_sampling processor")
	sf.ring = ext
}
