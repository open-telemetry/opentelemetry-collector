package sysreceiver

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

type sysReceiver struct {
	settings receiver.Settings
	config   *Config
	consumer consumer.Metrics
	cancel   context.CancelFunc
	wg       sync.WaitGroup

	scrapers []Scraper
}

func newSysReceiver(settings receiver.Settings, config *Config, consumer consumer.Metrics) (*sysReceiver, error) {
	return &sysReceiver{
		settings: settings,
		config:   config,
		consumer: consumer,
		// Initialize scrapers
		scrapers: []Scraper{
			newCpuScraper(),
			newMemScraper(),
			newDiskScraper(),
			newDiskIoScraper(),
			newNetScraper(),
		},
	}, nil
}

func (r *sysReceiver) Start(ctx context.Context, host component.Host) error {
	r.settings.Logger.Info("Starting system receiver",
		zap.String("node_value", r.config.NodeValue),
		zap.String("host_ip", r.config.HostIP))

	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	r.wg.Add(1)
	go r.startScrapeLoop(ctx)

	return nil
}

func (r *sysReceiver) Shutdown(ctx context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}
	r.wg.Wait()
	return nil
}

func (r *sysReceiver) startScrapeLoop(ctx context.Context) {
	defer r.wg.Done()

	ticker := time.NewTicker(r.config.CollectionInterval)
	defer ticker.Stop()

	// Initial scrape
	r.scrape(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.scrape(ctx)
		}
	}
}

func (r *sysReceiver) scrape(ctx context.Context) {
	r.settings.Logger.Debug("Starting system scrape")

	// 1. Prepare Metadata
	now := time.Now()
	meta := &Metadata{
		NodeValue:          r.config.NodeValue,
		NodeColumnName:     r.config.NodeColumnName,
		HostIP:             r.config.HostIP,
		DeploymentInstance: r.config.DeploymentInstance,
	}

	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()

	// Inject Resource Attributes (Standard)
	resAttrs := rm.Resource().Attributes()
	// Use dynamic Node attribute name + value
	resAttrs.PutStr(meta.NodeColumnName, meta.NodeValue)

	resAttrs.PutStr("host.ip", meta.HostIP)
	if meta.DeploymentInstance != "" {
		resAttrs.PutStr("deployment.instance", meta.DeploymentInstance)
	}

	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("sysreceiver")

	// 2. Run Scrapers
	for _, scraper := range r.scrapers {
		metrics, err := scraper.Scrape(r.settings.Logger, meta)
		if err != nil {
			r.settings.Logger.Error("Scraper failed", zap.Error(err))
			continue
		}

		for _, m := range metrics {
			dest := sm.Metrics().AppendEmpty()
			m.CopyTo(dest)
			// Ensure timestamp is set if not already
			if dest.Type() == pmetric.MetricTypeGauge {
				for i := 0; i < dest.Gauge().DataPoints().Len(); i++ {
					dp := dest.Gauge().DataPoints().At(i)
					if dp.Timestamp() == 0 {
						dp.SetTimestamp(pcommon.NewTimestampFromTime(now))
					}
				}
			}
		}
	}

	// 3. Export
	if sm.Metrics().Len() > 0 {
		if err := r.consumer.ConsumeMetrics(ctx, md); err != nil {
			r.settings.Logger.Error("Failed to consume metrics", zap.Error(err))
		} else {
			r.settings.Logger.Info("System metrics consumed", zap.Int("count", sm.Metrics().Len()))
		}
	}
}
