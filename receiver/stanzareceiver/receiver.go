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

package stanzareceiver

import (
	"context"
	"fmt"
	"sync"

	stanza "github.com/observiq/stanza/agent"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer"
)

type stanzareceiver struct {
	startOnce sync.Once
	stopOnce  sync.Once
	done      chan struct{}
	wg        sync.WaitGroup

	agent    *stanza.LogAgent
	emitter  *LogEmitter
	consumer consumer.LogsConsumer
	logger   *zap.Logger
}

// Ensure this factory adheres to required interface
var _ component.LogsReceiver = (*stanzareceiver)(nil)

// Start tells the receiver to start
func (r *stanzareceiver) Start(ctx context.Context, host component.Host) error {
	err := componenterror.ErrAlreadyStarted
	r.startOnce.Do(func() {
		err = nil
		r.logger.Info("Starting observiq receiver")

		if obsErr := r.agent.Start(); obsErr != nil {
			err = fmt.Errorf("start observiq: %s", err)
		}

		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			for {
				select {
				case <-r.done:
					return
				case obsLog := <-r.emitter.LogChan():
					if consumeErr := r.consumer.ConsumeLogs(ctx, convert(obsLog)); consumeErr != nil {
						r.logger.Error("ConsumeLogs() error", zap.String("error", consumeErr.Error()))
					}
				}
			}
		}()
	})

	return err
}

// Shutdown is invoked during service shutdown
func (r *stanzareceiver) Shutdown(context.Context) error {
	r.stopOnce.Do(func() {
		r.logger.Info("Shutting down observiq receiver")
		close(r.done)
		r.wg.Wait()
		r.emitter.Stop()
	})
	return nil
}
