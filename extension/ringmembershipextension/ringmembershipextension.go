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
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/extension"
)

type ringMembershipExtension struct {
	sync.RWMutex
	config  Config
	logger  *zap.Logger
	ipList  []string
	quit    chan int
	syncVar *chan interface{}
}

var _ (extension.SupportExtension) = (*ringMembershipExtension)(nil)
var _ (extension.ServiceExtension) = (*ringMembershipExtension)(nil)

// ByOrder implements sort.Interface for []string
type ByOrder []string

func (a ByOrder) Len() int           { return len(a) }
func (a ByOrder) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByOrder) Less(i, j int) bool { return a[i] < a[j] }

// Starts the ring membership extension
func (rm *ringMembershipExtension) Start(host extension.Host) error {
	rm.logger.Info("Starting ring membership extension")

	go rm.Watch()

	return nil
}

func (rm *ringMembershipExtension) Watch() error {
	t := time.NewTimer(10 * time.Millisecond)
	for {
		select {
		case <-rm.quit:
			return nil
		case <-t.C:
			// poll a dns endpoint
			ips, err := net.LookupIP("otelcol.svc.local")
			if err != nil {
				rm.logger.Error("DNS lookup error", zap.Error(err))
				return err
			}

			// check if this list has diverged from the current list
			var ipStrings []string
			for _, v := range ips {
				ipStrings = append(ipStrings, v.String())
			}

			// sort list of strings in-place
			sort.Sort(ByOrder(ipStrings))

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
				rm.logger.Info("Memberlist updated")
			}
		}
	}
}

func (rm *ringMembershipExtension) Shutdown() error {
	rm.quit <- -1
	return nil
}

func (rm *ringMembershipExtension) GetState() (interface{}, error) {
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
