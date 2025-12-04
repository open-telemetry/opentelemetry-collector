package rules

import (
	"context"
	"sync/atomic"
)

type RulesSource interface {
	WatchRulesUpdates() <-chan Rules
}

type AttributesMatcher interface {
	Match(attrs KVGetter, limit func(id int, count int) bool) *Rule
}

type RateLimiter interface {
	AllowNForRule(rule *Rule, n int) bool
	UpdateRules(rules Rules)
}

type Provider struct {
	processedRules atomic.Pointer[SSSRulesEngine] // *rules.SSSRulesEngine
	rulesSource    RulesSource
	rateLimiter    RateLimiter
}

func NewProvider(rulesSource RulesSource, rateLimiter RateLimiter, initialRules Rules) *Provider {
	p := &Provider{
		rulesSource: rulesSource,
		rateLimiter: rateLimiter,
	}
	p.processedRules.Store(NewSSSRulesEngine(initialRules))

	go p.updateRulesLoop(context.Background())

	return p
}

func (p *Provider) FindRule(attrs KVGetter) *Rule {
	rule := p.processedRules.Load().Match(attrs, p.rateLimiter.AllowNForRule)
	return rule
}

func (p *Provider) updateRulesLoop(ctx context.Context) {

	newRulesChan := p.rulesSource.WatchRulesUpdates()

	for {
		select {
		case newRules := <-newRulesChan:
			newEngine := NewSSSRulesEngine(newRules)
			p.processedRules.Store(newEngine)
		case <-ctx.Done():
			return
		}
	}
}
