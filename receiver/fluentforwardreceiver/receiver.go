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

package fluentforwardreceiver

import (
	"context"
	"net"
	"strings"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
)

// Give the event channel a bit of buffer to help reduce backpressure on
// FluentBit and increase throughput.
const eventChannelLength = 100

type fluentReceiver struct {
	collector *Collector
	listener  net.Listener
	conf      *Config
	logger    *zap.Logger
	server    *server
	cancel    context.CancelFunc
}

func newFluentReceiver(logger *zap.Logger, conf *Config, next consumer.LogsConsumer) (component.LogsReceiver, error) {
	eventCh := make(chan Event, eventChannelLength)

	collector := newCollector(eventCh, next, logger)

	server := newServer(eventCh, logger)

	return &fluentReceiver{
		collector: collector,
		server:    server,
		conf:      conf,
		logger:    logger,
	}, nil
}

func (r *fluentReceiver) Start(ctx context.Context, _ component.Host) error {
	receiverCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	r.collector.Start(receiverCtx)

	listenAddr := r.conf.ListenAddress

	var listener net.Listener
	var udpListener net.PacketConn
	var err error
	if strings.HasPrefix(listenAddr, "/") || strings.HasPrefix(listenAddr, "unix://") {
		listener, err = net.Listen("unix", strings.TrimPrefix(listenAddr, "unix://"))
	} else {
		listener, err = net.Listen("tcp", listenAddr)
		if err == nil {
			udpListener, err = net.ListenPacket("udp", listenAddr)
		}
	}

	if err != nil {
		return err
	}

	r.listener = listener

	r.server.Start(receiverCtx, listener)

	if udpListener != nil {
		go respondToHeartbeats(receiverCtx, udpListener, r.logger)
	}

	return nil
}

func (r *fluentReceiver) Shutdown(context.Context) error {
	r.cancel()
	return nil
}
