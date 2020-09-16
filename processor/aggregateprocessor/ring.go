// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aggregateprocessor

import (
	"net"

	"go.uber.org/zap"
)

type ringMemberQuerier interface {
	getMembers() []string
}

type dnsClient struct {
	// logger
	logger *zap.Logger
	// peerDiscoveryName
	peerDiscoveryName string
}

func newRingMemberQuerier(logger *zap.Logger, peerDiscoveryName string) ringMemberQuerier {
	return &dnsClient{
		logger:            logger,
		peerDiscoveryName: peerDiscoveryName,
	}
}

func (d *dnsClient) getMembers() []string {
	d.logger.Debug("Pooling a dns endpoint")
	// poll a dns endpoint
	ips, err := net.LookupIP(d.peerDiscoveryName)
	if err != nil {
		d.logger.Info("DNS lookup error", zap.Error(err))
		return nil
	}
	var newMembers []string
	for _, v := range ips {
		newMembers = append(newMembers, v.String())
	}
	return newMembers
}

var _ ringMemberQuerier = (*dnsClient)(nil)
