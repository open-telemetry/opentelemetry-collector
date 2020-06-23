package metrics

import (
	"go.opentelemetry.io/collector/internal/data"
	"go.opentelemetry.io/collector/testbed/testbed"
	"go.opentelemetry.io/collector/translator/internaldata"
)

type dualSender struct {
	mdsOld   testbed.MetricDataSenderOld
	mdsNew   testbed.MetricDataSender
	sendFunc func(data.MetricData) error
}

func NewDualSender(ds testbed.DataSender) *dualSender {
	switch s := ds.(type) {
	case testbed.MetricDataSender:
		out := &dualSender{mdsNew: s}
		out.sendFunc = out.sendNew
		return out
	case testbed.MetricDataSenderOld:
		out := &dualSender{mdsOld: s}
		out.sendFunc = out.sendOld
		return out
	}
	return nil
}

func (s *dualSender) send(md data.MetricData) error {
	return s.sendFunc(md)
}

func (s *dualSender) sendNew(md data.MetricData) error {
	return s.mdsNew.SendMetrics(md)
}

func (s *dualSender) sendOld(md data.MetricData) error {
	ocs := internaldata.MetricDataToOC(md)
	for _, oc := range ocs {
		err := s.mdsOld.SendMetrics(oc)
		if err != nil {
			return err
		}
	}
	return nil
}
