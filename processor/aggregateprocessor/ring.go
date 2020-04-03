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
