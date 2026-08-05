package sqlreceiver // import "go.opentelemetry.io/collector/receiver/sqlreceiver"

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

type sqlReceiver struct {
	config      *Config
	settings    receiver.Settings
	nextMetrics consumer.Metrics
	nextLogs    consumer.Logs
	client      *http.Client
	cancel      context.CancelFunc

	// Auth
	token    string
	tokenMux sync.RWMutex

	// Observability
	queryAttempt metric.Int64Counter
	queryFailure metric.Int64Counter
	rowsParsed   metric.Int64Counter
}

func newReceiver(config *Config, settings receiver.Settings, nextMetrics consumer.Metrics, nextLogs consumer.Logs) (*sqlReceiver, error) {
	r := &sqlReceiver{
		config:      config,
		settings:    settings,
		nextMetrics: nextMetrics,
		nextLogs:    nextLogs,
	}

	meter := settings.TelemetrySettings.MeterProvider.Meter("go.opentelemetry.io/collector/receiver/sqlreceiver")
	var err error

	if r.queryAttempt, err = meter.Int64Counter("sqlreceiver_query_attempt", metric.WithDescription("Number of query attempts")); err != nil {
		return nil, err
	}
	if r.queryFailure, err = meter.Int64Counter("sqlreceiver_query_failure", metric.WithDescription("Number of query failures")); err != nil {
		return nil, err
	}
	if r.rowsParsed, err = meter.Int64Counter("sqlreceiver_rows_parsed", metric.WithDescription("Number of rows successfully parsed")); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *sqlReceiver) Start(ctx context.Context, host component.Host) error {
	client, err := r.config.ClientConfig.ToClient(ctx, host.GetExtensions(), r.settings.TelemetrySettings)
	if err != nil {
		return err
	}
	r.client = client

	r.settings.Logger.Info("!!! SQL RECEIVER STARTING !!!",
		zap.Int("num_queries", len(r.config.Queries)),
		zap.String("endpoint", r.config.Endpoint))

	if len(r.config.Queries) > 0 {
		r.settings.Logger.Info("First Query Config", zap.Any("query_details", r.config.Queries[0]))
	} else {
		r.settings.Logger.Error("!!! NO QUERIES LOADED !!! Check your config.yaml indentation")
	}

	loopCtx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	if r.config.AuthEndpoint != "" {
		r.settings.Logger.Info("Auth endpoint configured, starting auth loop", zap.String("endpoint", r.config.AuthEndpoint))
		// Initial auth
		if err := r.refreshToken(ctx); err != nil {
			r.settings.Logger.Warn("Initial auth token fetch failed (will retry in loop)", zap.Error(err))
		}
		go r.authLoop(loopCtx)
	}

	for _, q := range r.config.Queries {
		r.settings.Logger.Info("Starting query loop", zap.String("query", q.Name))
		go r.startQueryLoop(loopCtx, q)
	}
	return nil
}

func (r *sqlReceiver) Shutdown(ctx context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}
	return nil
}

func (r *sqlReceiver) startQueryLoop(ctx context.Context, q *Query) {
	interval := r.config.CollectionInterval
	if interval <= 0 {
		interval = 10 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial scrape
	r.scrapeOfRetry(ctx, q)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.settings.Logger.Info("Ticker fired", zap.String("query", q.Name))
			r.scrapeOfRetry(ctx, q)
		}
	}
}

// Internal structures for parsing response
type innerResult struct {
	ColNames []map[string]string `json:"COL_NAMES"`
	Values   []map[string]string `json:"VALUES"`
}

type outerResponse struct {
	RetCode int    `json:"retcode"`
	RetMesg any    `json:"retmesg"`
	Result  string `json:"result"`
}

func (r *sqlReceiver) scrapeOfRetry(ctx context.Context, q *Query) {
	r.settings.Logger.Info("Scraping query", zap.String("query", q.Name))
	maxRetries := 3
	baseDelay := 1 * time.Second

	for i := 0; i <= maxRetries; i++ {
		r.queryAttempt.Add(ctx, 1, metric.WithAttributes(attribute.String("query_name", q.Name)))

		err := r.executeOnce(ctx, q)
		if err == nil {
			return // Success
		}

		// If context canceled, stop retry
		if ctx.Err() != nil {
			return
		}

		r.queryFailure.Add(ctx, 1, metric.WithAttributes(
			attribute.String("query_name", q.Name),
			attribute.String("error", err.Error()),
		))

		r.settings.Logger.Warn("Query failed, retrying",
			zap.String("query", q.Name),
			zap.Int("attempt", i+1),
			zap.Error(err))

		if i < maxRetries {
			delay := time.Duration(math.Pow(2, float64(i))) * baseDelay
			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return
			}
		}
	}
	r.settings.Logger.Error("Query failed after max retries", zap.String("query", q.Name))
}

func (r *sqlReceiver) executeOnce(ctx context.Context, q *Query) error {
	// r.settings.Logger.Info("Executing query", zap.String("name", q.Name)) // Too verbose for retry loops

	reqBody := map[string]string{"sql": q.SQL}
	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", r.config.Endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Inject Dynamic Token if available
	r.tokenMux.RLock()
	token := r.token
	r.tokenMux.RUnlock()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	r.settings.Logger.Info("Sending HTTP request", zap.String("query", q.Name), zap.String("endpoint", r.config.Endpoint))
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	r.settings.Logger.Info("HTTP response received", zap.String("query", q.Name), zap.String("status", resp.Status))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body failed: %w", err)
	}

	rows, err := r.parseResponse(body)
	if err != nil {
		return fmt.Errorf("parse response failed: %w", err)
	}

	r.rowsParsed.Add(ctx, int64(len(rows)), metric.WithAttributes(attribute.String("query_name", q.Name)))

	if q.SignalType == "logs" {
		if r.nextLogs != nil {
			ld := r.rowsToLogs(rows, q.Logs)
			if err := r.nextLogs.ConsumeLogs(ctx, ld); err != nil {
				return fmt.Errorf("consume logs failed: %w", err)
			}
		}
	} else {
		// Default to metrics
		if r.nextMetrics != nil {
			md := r.rowsToMetrics(rows, q.Metrics)
			if err := r.nextMetrics.ConsumeMetrics(ctx, md); err != nil {
				return fmt.Errorf("consume metrics failed: %w", err)
			}
		}
	}

	// r.settings.Logger.Info("Query executed successfully", zap.String("name", q.Name))
	return nil
}

func (r *sqlReceiver) parseResponse(body []byte) ([]map[string]string, error) {
	var outer outerResponse
	if err := json.Unmarshal(body, &outer); err != nil {
		return nil, fmt.Errorf("outer json unmarshal failed: %w", err)
	}

	if outer.RetCode != 0 {
		return nil, fmt.Errorf("api returned error code: %d, msg: %v", outer.RetCode, outer.RetMesg)
	}

	var inner innerResult
	if err := json.Unmarshal([]byte(outer.Result), &inner); err != nil {
		return nil, fmt.Errorf("inner result json unmarshal failed: %w", err)
	}

	return inner.Values, nil
}

func (r *sqlReceiver) rowsToMetrics(rows []map[string]string, metrics []*MetricConfig) pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()

	r.settings.Logger.Info("rowsToMetrics processing", zap.Int("rows_count", len(rows)))
	if len(rows) > 0 {
		r.settings.Logger.Info("First row content", zap.Any("row", rows[0]))
	}

	for _, row := range rows {
		// Single row can produce multiple metrics
		for _, mapping := range metrics {
			valStr, ok := row[mapping.ValueField]
			if !ok {
				r.settings.Logger.Warn("ValueField not found in row",
					zap.String("field", mapping.ValueField),
					zap.Any("row_keys", row))
				continue
			}

			cleanVal := strings.TrimSpace(valStr)
			cleanVal = strings.TrimSuffix(cleanVal, "%")
			cleanVal = strings.TrimSpace(cleanVal)

			val, err := strconv.ParseFloat(cleanVal, 64)
			if err != nil {
				r.settings.Logger.Warn("Failed to parse metric value",
					zap.String("metric", mapping.Name),
					zap.String("value", valStr),
					zap.Error(err))
				continue
			}

			// Log success for the first few metrics to verify flow
			// r.settings.Logger.Info("Metric generated", zap.String("name", mapping.Name), zap.Float64("val", val))

			m := sm.Metrics().AppendEmpty()
			m.SetName(mapping.Name)

			var dp pmetric.NumberDataPoint
			switch mapping.MetricType {
			case "sum":
				m.SetEmptySum().SetIsMonotonic(true)
				m.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
				dp = m.Sum().DataPoints().AppendEmpty()
			default:
				m.SetEmptyGauge()
				dp = m.Gauge().DataPoints().AppendEmpty()
			}

			dp.SetDoubleValue(val)
			dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

			// [FORCE INJECT] Identity Attributes to ensure they appear as columns in GreptimeDB
			// 1. host.name
			if hostname, err := os.Hostname(); err == nil {
				dp.Attributes().PutStr("host.name", hostname)
			}
			// 2. service.name (Hardcoded as requested to ensure consistency)
			dp.Attributes().PutStr("service.name", "sundb")

			// 3. deployment.instance (Configurable)
			if r.config.DeploymentInstance != "" {
				dp.Attributes().PutStr("deployment.instance", r.config.DeploymentInstance)
			}

			for dbField, attrName := range mapping.Attributes {
				if v, ok := row[dbField]; ok {
					dp.Attributes().PutStr(attrName, v)
				}
			}
		}
	}
	return md
}

func (r *sqlReceiver) rowsToLogs(rows []map[string]string, logConfig *LogConfig) plog.Logs {
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()

	if logConfig == nil {
		return ld
	}

	for _, row := range rows {
		lr := sl.LogRecords().AppendEmpty()
		lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))

		// Set Body
		if v, ok := row[logConfig.BodyField]; ok {
			lr.Body().SetStr(v)
		} else {
			// fallback: serialize entire row
			bodyBytes, _ := json.Marshal(row)
			lr.Body().SetStr(string(bodyBytes))
		}

		// Attributes
		for dbField, attrName := range logConfig.Attributes {
			if v, ok := row[dbField]; ok {
				lr.Attributes().PutStr(attrName, v)
			}
		}
	}
	return ld
}

func (r *sqlReceiver) authLoop(ctx context.Context) {
	// 1. Startup Retry Loop
	// If token is missing, retry frequently until successful
	retryTicker := time.NewTicker(30 * time.Second)
	defer retryTicker.Stop()

	for {
		r.tokenMux.RLock()
		hasToken := r.token != ""
		r.tokenMux.RUnlock()

		if hasToken {
			break
		}

		if err := r.refreshToken(ctx); err == nil {
			break
		}

		select {
		case <-ctx.Done():
			return
		case <-retryTicker.C:
			continue
		}
	}

	// 2. Regular Refresh Loop (6 Hours)
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.refreshToken(ctx); err != nil {
				r.settings.Logger.Error("Auth token refresh failed", zap.Error(err))
			}
		}
	}
}

func (r *sqlReceiver) refreshToken(ctx context.Context) error {
	r.settings.Logger.Info("Refreshing auth token", zap.String("endpoint", r.config.AuthEndpoint))

	data := url.Values{}
	data.Set("password", r.config.AuthPassword)

	req, err := http.NewRequestWithContext(ctx, "POST", r.config.AuthEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("auth failed: %s, body: %s", resp.Status, string(body))
	}

	var res struct {
		Token string `json:"token"`
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return err
	}

	if res.Token == "" {
		return fmt.Errorf("token empty in response")
	}

	r.tokenMux.Lock()
	r.token = res.Token
	r.tokenMux.Unlock()
	r.settings.Logger.Info("Auth token refreshed successfully")
	return nil
}
