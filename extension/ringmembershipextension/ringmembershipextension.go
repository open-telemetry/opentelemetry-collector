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

package ringmembershipextension

import (
	"net"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/facette/natsort"
	"github.com/open-telemetry/opentelemetry-collector/extension"
)

type ringMembershipExtension struct {
	sync.RWMutex
	config Config
	logger *zap.Logger
	ipList []string
	quit   chan int
}

var _ (extension.SupportExtension) = (*ringMembershipExtension)(nil)
var _ (extension.ServiceExtension) = (*ringMembershipExtension)(nil)

// Starts the ring membership extension
func (rm *ringMembershipExtension) Start(host extension.Host) error {
	rm.logger.Info("Starting ring membership extension")

	rm.quit = make(chan int, 1)

	go rm.Watch()

	return nil
}

func (rm *ringMembershipExtension) Watch() error {
	t := time.NewTicker(time.Second)
	for {
		select {
		case <-rm.quit:
			return nil
		case <-t.C:
			rm.logger.Debug("Pooling a dns endpoint")
			// poll a dns endpoint
			ips, err := net.LookupIP("otelcol-headless.default.svc.cluster.local.")
			if err != nil {
				rm.logger.Error("DNS lookup error", zap.Error(err))
				break
			}

			// check if this list has diverged from the current list
			var ipStrings []string
			for _, v := range ips {
				ipStrings = append(ipStrings, v.String())
			}

			natsort.Sort(ipStrings)

			// check if its the same as the current member list
			rm.RLock()
			curList := rm.ipList
			rm.RUnlock()

			isEqual := true
			if len(curList) != len(ipStrings) {
				isEqual = false
			} else {
				for k, v := range curList {
					if v != ipStrings[k] {
						isEqual = false
					}
				}
			}

			if !isEqual {
				// Update state
				rm.Lock()
				rm.ipList = ipStrings
				rm.Unlock()
				rm.logger.Info("Memberlist updated", zap.Strings("Members", ipStrings))
			}
		default:
			// rm.logger.Info("Hitting default case")
		}
	}
}

func (rm *ringMembershipExtension) Shutdown() error {
	rm.quit <- -1
	return nil
}

func (rm *ringMembershipExtension) GetState() (interface{}, error) {
	rm.logger.Debug("GetState called in ringmembershipextension")
	rm.RLock()
	curList := rm.ipList
	rm.RUnlock()

	return curList, nil
}

func newServer(config Config, logger *zap.Logger) (*ringMembershipExtension, error) {
	rm := &ringMembershipExtension{
		config: config,
		logger: logger,
	}

	return rm, nil
}
