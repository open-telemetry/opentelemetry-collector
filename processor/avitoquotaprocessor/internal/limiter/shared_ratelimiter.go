package limiter

import (
	"context"
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"

	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/limiter/rate"
	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/metadata"
	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/rules"
	"go.uber.org/zap"
)

// entry holds limiter and metadata.
type entry struct {
	limiter  *rate.Limiter
	r        rate.Limit
	burst    int
	lastSeen int64 // unix nano
}

// shard contains a segment of the global map.
type shard struct {
	mu    sync.Mutex
	m     map[string]*entry
	count int64 // number of entries (mirror of len(m)) for cheap read
}

// ShardedManager is a per-key rate limiter manager using sharded maps.
// It minimizes contention by hashing keys into shards.
type ShardedManager struct {
	shards        []*shard
	numShards     uint32
	cleanupAfter  time.Duration
	cleanupTicker *time.Ticker
	closed        chan struct{}
	// stats
	totalEntries int64 // approximate
	telemetry    *metadata.TelemetryBuilder
	logger       *zap.Logger
	dryRun       bool
}

// DefaultShardCount recommended for most systems. Use a power of two.
const DefaultShardCount = 32
const RateLimiterName = "shared-rate-limiter"

const EverySecoundsRL = "sec_"
const EveryDayRL = "day_"

// NewShardedManager creates a manager with the given number of shards.
// cleanupAfter - duration of inactivity after which entries are evicted (0 disables eviction).
// cleanupInterval - how often the janitor runs (ignored if cleanupAfter==0).
// If numShards <= 0, DefaultShardCount is used.
func NewShardedManager(telemetry *metadata.TelemetryBuilder, logger *zap.Logger, dryRun bool) *ShardedManager {
	numShards := DefaultShardCount
	cleanupAfter := 0 * time.Minute
	//cleanupInterval := 0 * time.Minute

	numShards = DefaultShardCount
	// ensure power of two for fast masking if desired, but not strictly necessary
	shards := make([]*shard, numShards)
	for i := 0; i < numShards; i++ {
		shards[i] = &shard{m: make(map[string]*entry)}
	}
	m := &ShardedManager{
		shards:       shards,
		numShards:    uint32(numShards),
		cleanupAfter: cleanupAfter,
		closed:       make(chan struct{}),
		telemetry:    telemetry,
		logger:       logger,
		dryRun:       dryRun,
	}
	//if cleanupAfter > 0 && cleanupInterval > 0 {
	//	m.cleanupTicker = time.NewTicker(cleanupInterval)
	//	go m.janitorLoop()
	//}
	return m
}

// Close stops background janitor. Safe to call multiple times.
func (m *ShardedManager) Close() {
	select {
	case <-m.closed:
		// already closed
		return
	default:
		close(m.closed)
		if m.cleanupTicker != nil {
			m.cleanupTicker.Stop()
		}
	}
}

// hashKey returns a uint32 hash of the key.
func hashKey(key string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return h.Sum32()
}

// pickShard returns the shard for the key.
func (m *ShardedManager) pickShard(key string) *shard {
	h := hashKey(key)
	idx := uint32(h) % m.numShards
	return m.shards[idx]
}

// nowUnixNano helper.
func nowUnixNano() int64 { return time.Now().UnixNano() }

// getEntryLocked returns entry and bool under the shard lock. Caller must hold shard.mu.
func (s *shard) getEntryLocked(key string) (*entry, bool) {
	e, ok := s.m[key]
	return e, ok
}

// createOrUpdateEntryLocked creates a new entry or returns existing. Caller must hold shard.mu.
func (s *shard) createOrUpdateEntryLocked(key string, r rate.Limit, burst int) *entry {
	if e, ok := s.m[key]; ok {
		// if params changed, replace limiter
		if e.r != r || e.burst != burst {
			// create new limiter preserving no accumulated tokens (start full)
			newL := rate.NewLimiter(r, burst)
			e.limiter = newL
			e.r = r
			e.burst = burst
		}
		e.lastSeen = nowUnixNano()
		return e
	}
	// create new
	ent := &entry{
		limiter:  rate.NewLimiter(r, burst),
		r:        r,
		burst:    burst,
		lastSeen: nowUnixNano(),
	}
	s.m[key] = ent
	atomic.AddInt64(&s.count, 1)
	return ent
}

// GetLimiterFor returns the *rate.Limiter for the key, creating it if necessary.
// This is the hot path; it uses sharding and fine-grained locking.
func (m *ShardedManager) GetLimiterFor(key string, r rate.Limit, burst int) *rate.Limiter {
	s := m.pickShard(key)
	s.mu.Lock()
	ent, ok := s.getEntryLocked(key)
	if ok {
		ent.lastSeen = nowUnixNano()
		lim := ent.limiter
		s.mu.Unlock()
		return lim
	}
	// create
	ent = s.createOrUpdateEntryLocked(key, r, burst)
	lim := ent.limiter
	s.mu.Unlock()
	atomic.AddInt64(&m.totalEntries, 1)
	return lim
}

// ReserveN reserves n tokens; returns delay and ok.
func (m *ShardedManager) ReserveN(key string, r rate.Limit, burst, n int) (time.Duration, bool) {
	lim := m.GetLimiterFor(key, r, burst)
	res := lim.ReserveN(time.Now(), n)
	if !res.OK() {
		return 0, false
	}
	return res.DelayFrom(time.Now()), true
}

// UpdateLimit forcibly changes r/burst for a key (creating the entry if missing).
// This replaces the limiter instance immediately; outstanding reservations on the
// old limiter will not transfer. Use cautiously.
func (m *ShardedManager) UpdateLimit(key string, r rate.Limit, burst int) {
	s := m.pickShard(key)
	s.mu.Lock()
	if e, ok := s.m[key]; ok {
		// replace limiter
		e.limiter = rate.NewLimiter(r, burst)
		e.r = r
		e.burst = burst
		e.lastSeen = nowUnixNano()
		s.mu.Unlock()
		return
	}
	// create
	s.m[key] = &entry{
		limiter:  rate.NewLimiter(r, burst),
		r:        r,
		burst:    burst,
		lastSeen: nowUnixNano(),
	}
	atomic.AddInt64(&s.count, 1)
	s.mu.Unlock()
	atomic.AddInt64(&m.totalEntries, 1)
}

// janitorLoop periodically scans shards and evicts entries idle longer than cleanupAfter.
func (m *ShardedManager) janitorLoop() {
	if m.cleanupTicker == nil {
		return
	}
	for {
		select {
		case <-m.cleanupTicker.C:
			now := time.Now().UnixNano()
			threshold := now - int64(m.cleanupAfter)
			var removed int64
			for _, s := range m.shards {
				s.mu.Lock()
				for k, e := range s.m {
					if e.lastSeen < threshold {
						delete(s.m, k)
						removed++
						atomic.AddInt64(&s.count, -1)
					}
				}
				s.mu.Unlock()
			}
			if removed > 0 {
				// approximate adjust
				atomic.AddInt64(&m.totalEntries, -removed)
			}
		case <-m.closed:
			return
		}
	}
}

// Purge removes all entries from every shard. Useful for tests.
func (m *ShardedManager) Purge() {
	for _, s := range m.shards {
		s.mu.Lock()
		for k := range s.m {
			delete(s.m, k)
		}
		atomic.StoreInt64(&s.count, 0)
		s.mu.Unlock()
	}
	atomic.StoreInt64(&m.totalEntries, 0)
}

func (m *ShardedManager) Allow(key string, r rate.Limit, burst int) bool {
	lim := m.GetLimiterFor(key, r, burst)
	return lim.Allow()
}

// AllowForRule check 1 token for Rule
func (m *ShardedManager) AllowForRule(rule *rules.Rule) bool {
	return m.AllowNForRule(rule, 1)
}

// AllowNForRule check N tokens for rule
func (m *ShardedManager) AllowNForRule(rule *rules.Rule, n int) bool {
	limRatePerSecond := m.GetLimiterFor(EverySecoundsRL+rule.ID, rate.Every(time.Second/time.Duration(rule.RatePerSecond)), rule.RatePerSecond*2)
	limRatePerDay := m.GetLimiterFor(EveryDayRL+rule.ID, rate.Every(24*time.Hour/time.Duration(rule.StorageRatePerDay)), rule.StorageRatePerDay*2)

	allowed := limRatePerSecond.Tokens() > float64(n) && limRatePerDay.Tokens() > float64(n)
	if allowed {
		limRatePerSecond.AllowN(time.Now(), n)
		limRatePerDay.AllowN(time.Now(), n)
		incRuleAttempt(context.Background(), m.telemetry, rule.Name, attemptResultOk, RateLimiterName, int64(n))
	} else {
		m.logger.Warn(
			fmt.Sprintf(
				"rate limited. per_sec tokens: %f, per_day tokens: %f",
				limRatePerSecond.Tokens(),
				limRatePerDay.Tokens(),
			),
		)
		incRuleAttempt(context.Background(), m.telemetry, rule.Name, attemptResultRateLimited, RateLimiterName, int64(n))
	}
	return allowed || m.dryRun
}

func (m *ShardedManager) UpdateRules(rules rules.Rules) {
	for _, rule := range rules {
		m.logger.Info(fmt.Sprintf("Updating rule %s with limit %d", rule.Name, rule.RatePerSecond))
		m.UpdateRuleLimit(rule)
	}
}

func (m *ShardedManager) UpdateRuleLimit(rule *rules.Rule) {
	m.UpdateLimit(rule.ID, rate.Limit(rule.RatePerSecond), rule.RatePerSecond*2)
}
