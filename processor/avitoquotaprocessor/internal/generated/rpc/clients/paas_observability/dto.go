// Code generated - DO NOT EDIT.
package paas_observability

type ListLogsRulesIn struct {
}

type LogsAttributeKeyV3 struct {
	Name string `json:"name"` // Ключ атрибута, например "service" или "severity"
	Kind string `json:"kind"` // Вид атрибута, принимает одно из значений AttributeKind, (log, resource, system)
}

func (s *LogsAttributeKeyV3) GetName() string      { return s.Name }
func (s *LogsAttributeKeyV3) SetName(value string) { s.Name = value }
func (s *LogsAttributeKeyV3) GetKind() string      { return s.Kind }
func (s *LogsAttributeKeyV3) SetKind(value string) { s.Kind = value }

type LogsAttributeExpression struct {
	Key      LogsAttributeKeyV3 `json:"key"`
	Operator *string            `json:"operator,omitempty"` // Оператор фильтрации. Один из LogTagOperator. Отсутствие приравнивается к "="
	Value    *string            `json:"value,omitempty"`    // Значение фильтра, поддерживается всеми операторами, кроме IN, NOT_IN
	Values   *[]string          `json:"values,omitempty"`   // Множественное значение фильтра, поддерживается только операторами IN, NOT_IN
}

func (s *LogsAttributeExpression) GetKey() LogsAttributeKeyV3      { return s.Key }
func (s *LogsAttributeExpression) SetKey(value LogsAttributeKeyV3) { s.Key = value }
func (s *LogsAttributeExpression) GetOperator() *string            { return s.Operator }
func (s *LogsAttributeExpression) SetOperator(value *string)       { s.Operator = value }
func (s *LogsAttributeExpression) GetValue() *string               { return s.Value }
func (s *LogsAttributeExpression) SetValue(value *string)          { s.Value = value }
func (s *LogsAttributeExpression) GetValues() *[]string            { return s.Values }
func (s *LogsAttributeExpression) SetValues(value *[]string)       { s.Values = value }

type LogsRuleVersionQuota struct {
	ResourceMetricID string `json:"resourceMetricID"`
	Value            int64  `json:"value"`
}

func (s *LogsRuleVersionQuota) GetResourceMetricID() string      { return s.ResourceMetricID }
func (s *LogsRuleVersionQuota) SetResourceMetricID(value string) { s.ResourceMetricID = value }
func (s *LogsRuleVersionQuota) GetValue() int64                  { return s.Value }
func (s *LogsRuleVersionQuota) SetValue(value int64)             { s.Value = value }

type LogsRuleTTLProfile struct {
	Id              string `json:"id"`
	ResourceId      string `json:"resourceId"`
	Name            string `json:"name"`
	DurationSeconds int64  `json:"durationSeconds"`
	IsDefault       bool   `json:"isDefault"`
}

func (s *LogsRuleTTLProfile) GetId() string                  { return s.Id }
func (s *LogsRuleTTLProfile) SetId(value string)             { s.Id = value }
func (s *LogsRuleTTLProfile) GetResourceId() string          { return s.ResourceId }
func (s *LogsRuleTTLProfile) SetResourceId(value string)     { s.ResourceId = value }
func (s *LogsRuleTTLProfile) GetName() string                { return s.Name }
func (s *LogsRuleTTLProfile) SetName(value string)           { s.Name = value }
func (s *LogsRuleTTLProfile) GetDurationSeconds() int64      { return s.DurationSeconds }
func (s *LogsRuleTTLProfile) SetDurationSeconds(value int64) { s.DurationSeconds = value }
func (s *LogsRuleTTLProfile) GetIsDefault() bool             { return s.IsDefault }
func (s *LogsRuleTTLProfile) SetIsDefault(value bool)        { s.IsDefault = value }

type ActualLogsRule struct {
	RuleID   string                    `json:"ruleID"`
	RuleName string                    `json:"ruleName"`
	Filter   []LogsAttributeExpression `json:"filter"`
	Quotas   []LogsRuleVersionQuota    `json:"quotas"`
	Ttl      LogsRuleTTLProfile        `json:"ttl"`
}

func (s *ActualLogsRule) GetRuleID() string                         { return s.RuleID }
func (s *ActualLogsRule) SetRuleID(value string)                    { s.RuleID = value }
func (s *ActualLogsRule) GetRuleName() string                       { return s.RuleName }
func (s *ActualLogsRule) SetRuleName(value string)                  { s.RuleName = value }
func (s *ActualLogsRule) GetFilter() []LogsAttributeExpression      { return s.Filter }
func (s *ActualLogsRule) SetFilter(value []LogsAttributeExpression) { s.Filter = value }
func (s *ActualLogsRule) GetQuotas() []LogsRuleVersionQuota         { return s.Quotas }
func (s *ActualLogsRule) SetQuotas(value []LogsRuleVersionQuota)    { s.Quotas = value }
func (s *ActualLogsRule) GetTtl() LogsRuleTTLProfile                { return s.Ttl }
func (s *ActualLogsRule) SetTtl(value LogsRuleTTLProfile)           { s.Ttl = value }

type ListLogsRulesOut struct {
	Rules []ActualLogsRule `json:"rules"`
}

func (s *ListLogsRulesOut) GetRules() []ActualLogsRule      { return s.Rules }
func (s *ListLogsRulesOut) SetRules(value []ActualLogsRule) { s.Rules = value }
