// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

type SendingQueueSupport string

const (
	SendingQueueSupportDefault      SendingQueueSupport = "default"
	SendingQueueSupportHasOverrides SendingQueueSupport = "has_overrides"
	SendingQueueSupportOmitted      SendingQueueSupport = "omitted"
)

type SendingQueue struct {
	Support   SendingQueueSupport   `mapstructure:"support"`
	Rationale string                `mapstructure:"rationale"`
	Overrides SendingQueueOverrides `mapstructure:"overrides"`
}

type SendingQueueOverrides map[string]any

type parsedSendingQueueOverrides struct {
	enabled             bool
	enabledPresent      bool
	queue               map[string]any
	batchPresent        bool
	batchEnabled        bool
	batchEnabledPresent bool
	batchOverrides      map[string]any
}

// SendingQueueTemplateData contains values rendered by the sending queue template.
type SendingQueueTemplateData struct {
	QueueEnabled       bool
	WaitForResult      bool
	QueueSizer         string
	QueueSizerValue    string
	QueueSize          int64
	BlockOnOverflow    bool
	StorageConstructor string
	StorageValue       string
	NumConsumers       int
	BatchEnabled       bool
	FlushTimeout       int64
	BatchSizer         string
	BatchSizerValue    string
	MinSize            int64
	MaxSize            int64
	MetadataKeys       string
	MetadataKeyValues  []string
}

type sendingQueueDocumentation struct {
	SendingQueue queueDocumentation `yaml:"sending_queue"`
}

type queueDocumentation struct {
	Enabled         bool               `yaml:"enabled"`
	WaitForResult   bool               `yaml:"wait_for_result"`
	Sizer           string             `yaml:"sizer"`
	QueueSize       int64              `yaml:"queue_size"`
	BlockOnOverflow bool               `yaml:"block_on_overflow"`
	Storage         *string            `yaml:"storage"`
	NumConsumers    int                `yaml:"num_consumers"`
	Batch           batchDocumentation `yaml:"batch"`
}

type batchDocumentation struct {
	Enabled      bool                   `yaml:"enabled"`
	FlushTimeout string                 `yaml:"flush_timeout"`
	Sizer        string                 `yaml:"sizer"`
	MinSize      int64                  `yaml:"min_size"`
	MaxSize      int64                  `yaml:"max_size"`
	Partition    partitionDocumentation `yaml:"partition"`
}

type partitionDocumentation struct {
	MetadataKeys []string `yaml:"metadata_keys"`
}

func (sq *SendingQueue) Validate() error {
	if sq.Support == "" {
		sq.Support = SendingQueueSupportDefault
	}
	if err := sq.validateDeclaration(); err != nil {
		return err
	}
	if sq.IsOmitted() {
		return nil
	}

	base := exporterhelper.NewDefaultQueueConfig()
	if sq.HasOverrides() {
		base = newPostMigrationDefaultQueueConfig()
	}
	cfg, err := sq.Overrides.Apply(base)
	if err != nil {
		return err
	}
	if err := confmap.Validate(cfg); err != nil {
		return fmt.Errorf("invalid sending_queue.overrides: %w", err)
	}
	return nil
}

func (sq *SendingQueue) validateDeclaration() error {
	switch sq.Support {
	case SendingQueueSupportDefault:
		if len(sq.Overrides) != 0 {
			return errors.New("sending_queue.overrides requires support to be has_overrides")
		}
	case SendingQueueSupportHasOverrides:
		if len(sq.Overrides) == 0 {
			return errors.New("sending_queue.overrides is required when support is has_overrides")
		}
	case SendingQueueSupportOmitted:
		var errs error
		if strings.TrimSpace(sq.Rationale) == "" {
			errs = errors.Join(errs, errors.New("sending_queue.rationale is required when support is omitted"))
		}
		if len(sq.Overrides) != 0 {
			errs = errors.Join(errs, errors.New("sending_queue.overrides cannot be set when support is omitted"))
		}
		return errs
	default:
		return fmt.Errorf("sending_queue.support must be one of %q, %q, or %q", SendingQueueSupportDefault, SendingQueueSupportHasOverrides, SendingQueueSupportOmitted)
	}

	parsed, err := sq.Overrides.parse()
	if err != nil {
		return err
	}
	if sq.HasOverrides() {
		var errs error
		if !parsed.enabledPresent {
			errs = errors.Join(errs, errors.New("sending_queue.overrides.enabled is required when support is has_overrides"))
		}
		if !parsed.batchEnabledPresent {
			errs = errors.Join(errs, errors.New("sending_queue.overrides.batch.enabled is required when support is has_overrides"))
		}
		if errs != nil {
			return errs
		}
	}
	if !parsed.enabled && strings.TrimSpace(sq.Rationale) == "" {
		return errors.New("sending_queue.rationale is required when overrides disable the queue")
	}
	return nil
}

func (sq *SendingQueue) IsOmitted() bool {
	return sq.Support == SendingQueueSupportOmitted
}

func (sq *SendingQueue) HasOverrides() bool {
	return sq.Support == SendingQueueSupportHasOverrides
}

func (sq *SendingQueue) HasOverride(path ...string) bool {
	return hasOverride(sq.Overrides, path)
}

func (sq *SendingQueue) TemplateData() (SendingQueueTemplateData, error) {
	optionalCfg, err := sq.Overrides.Apply(newPostMigrationDefaultQueueConfig())
	if err != nil {
		return SendingQueueTemplateData{}, err
	}
	queueEnabled := optionalCfg.HasValue()
	cfg := optionalCfg.GetOrInsertDefault()
	batchEnabled := cfg.Batch.HasValue()
	batchCfg := *cfg.Batch.GetOrInsertDefault()

	queueSizer, err := sizerExpression(cfg.Sizer.String())
	if err != nil {
		return SendingQueueTemplateData{}, err
	}
	batchSizer, err := sizerExpression(batchCfg.Sizer.String())
	if err != nil {
		return SendingQueueTemplateData{}, err
	}

	storageConstructor := ""
	storageValue := ""
	if cfg.StorageID != nil {
		constructor := "component.MustNewID"
		args := strconv.Quote(cfg.StorageID.Type().String())
		if cfg.StorageID.Name() != "" {
			constructor = "component.MustNewIDWithName"
			args += ", " + strconv.Quote(cfg.StorageID.Name())
		}
		storageConstructor = constructor + "(" + args + ")"
		storageValue = cfg.StorageID.String()
	}

	keys := make([]string, 0, len(batchCfg.Partition.MetadataKeys))
	for _, key := range batchCfg.Partition.MetadataKeys {
		keys = append(keys, strconv.Quote(key))
	}

	return SendingQueueTemplateData{
		QueueEnabled:       queueEnabled,
		WaitForResult:      cfg.WaitForResult,
		QueueSizer:         queueSizer,
		QueueSizerValue:    cfg.Sizer.String(),
		QueueSize:          cfg.QueueSize,
		BlockOnOverflow:    cfg.BlockOnOverflow,
		StorageConstructor: storageConstructor,
		StorageValue:       storageValue,
		NumConsumers:       cfg.NumConsumers,
		BatchEnabled:       batchEnabled,
		FlushTimeout:       int64(batchCfg.FlushTimeout),
		BatchSizer:         batchSizer,
		BatchSizerValue:    batchCfg.Sizer.String(),
		MinSize:            batchCfg.MinSize,
		MaxSize:            batchCfg.MaxSize,
		MetadataKeys:       "[]string{" + strings.Join(keys, ", ") + "}",
		MetadataKeyValues:  batchCfg.Partition.MetadataKeys,
	}, nil
}

func (sq *SendingQueue) YAMLConfig() (string, error) {
	templateData, err := sq.TemplateData()
	if err != nil {
		return "", err
	}

	var storage *string
	if templateData.StorageValue != "" {
		storage = &templateData.StorageValue
	}
	doc := sendingQueueDocumentation{
		SendingQueue: queueDocumentation{
			Enabled:         templateData.QueueEnabled,
			WaitForResult:   templateData.WaitForResult,
			Sizer:           templateData.QueueSizerValue,
			QueueSize:       templateData.QueueSize,
			BlockOnOverflow: templateData.BlockOnOverflow,
			Storage:         storage,
			NumConsumers:    templateData.NumConsumers,
			Batch: batchDocumentation{
				Enabled:      templateData.BatchEnabled,
				FlushTimeout: time.Duration(templateData.FlushTimeout).String(),
				Sizer:        templateData.BatchSizerValue,
				MinSize:      templateData.MinSize,
				MaxSize:      templateData.MaxSize,
				Partition: partitionDocumentation{
					MetadataKeys: templateData.MetadataKeyValues,
				},
			},
		},
	}
	var node yaml.Node
	if encodeErr := node.Encode(doc); encodeErr != nil {
		return "", encodeErr
	}
	annotateSendingQueueLeaves(&node, nil, sq)

	encoded, err := yaml.Marshal(&node)
	if err != nil {
		return "", err
	}
	return alignYAMLComments(strings.TrimSpace(string(encoded))), nil
}

func annotateSendingQueueLeaves(node *yaml.Node, path []string, sq *SendingQueue) {
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			key, value := node.Content[i], node.Content[i+1]
			annotateSendingQueueLeaves(value, append(path, key.Value), sq)
		}
	case yaml.ScalarNode, yaml.SequenceNode:
		overridePath := path
		if len(overridePath) > 0 && overridePath[0] == "sending_queue" {
			overridePath = overridePath[1:]
		}
		switch {
		case sq.Support == SendingQueueSupportDefault && slices.Equal(overridePath, []string{"batch", "enabled"}):
			node.LineComment = "FEATURE(pkg.exporterhelper.queueBatchEnabled)"
		case hasOverride(sq.Overrides, overridePath):
			node.LineComment = "OVERRIDE"
		default:
			node.LineComment = "default"
		}
	}
}

func hasOverride(overrides map[string]any, path []string) bool {
	current := overrides
	for i, key := range path {
		value, ok := current[key]
		if !ok {
			return false
		}
		if i == len(path)-1 {
			return true
		}
		current, ok = value.(map[string]any)
		if !ok {
			return false
		}
	}
	return false
}

func alignYAMLComments(yamlConfig string) string {
	lines := strings.Split(yamlConfig, "\n")
	maxContentWidth := 0
	for _, line := range lines {
		if content, _, ok := strings.Cut(line, " # "); ok {
			maxContentWidth = max(maxContentWidth, len(strings.TrimRight(content, " ")))
		}
	}
	for i, line := range lines {
		content, comment, ok := strings.Cut(line, " # ")
		if !ok {
			continue
		}
		content = strings.TrimRight(content, " ")
		lines[i] = content + strings.Repeat(" ", maxContentWidth-len(content)+2) + "# " + comment
	}
	return strings.Join(lines, "\n")
}

func (overrides SendingQueueOverrides) Apply(cfg exporterhelper.QueueBatchConfig) (configoptional.Optional[exporterhelper.QueueBatchConfig], error) {
	parsed, err := overrides.parse()
	if err != nil {
		return configoptional.None[exporterhelper.QueueBatchConfig](), err
	}

	if err := confmap.NewFromStringMap(parsed.queue).Unmarshal(&cfg); err != nil {
		return configoptional.None[exporterhelper.QueueBatchConfig](), fmt.Errorf("invalid sending_queue.overrides: %w", err)
	}
	if parsed.batchPresent {
		batchCfg := *cfg.Batch.GetOrInsertDefault()
		if err := confmap.NewFromStringMap(parsed.batchOverrides).Unmarshal(&batchCfg); err != nil {
			return configoptional.None[exporterhelper.QueueBatchConfig](), fmt.Errorf("invalid sending_queue.overrides.batch: %w", err)
		}
		if parsed.batchEnabled {
			cfg.Batch = configoptional.Some(batchCfg)
		} else {
			cfg.Batch = configoptional.Default(batchCfg)
		}
	}
	if parsed.enabled {
		return configoptional.Some(cfg), nil
	}
	return configoptional.Default(cfg), nil
}

func newPostMigrationDefaultQueueConfig() exporterhelper.QueueBatchConfig {
	cfg := exporterhelper.NewDefaultQueueConfig()
	batchCfg := *cfg.Batch.GetOrInsertDefault()
	cfg.Batch = configoptional.Some(batchCfg)
	return cfg
}

func (overrides SendingQueueOverrides) parse() (parsedSendingQueueOverrides, error) {
	parsed := parsedSendingQueueOverrides{
		enabled: true,
		queue:   make(map[string]any, len(overrides)),
	}
	for key, value := range overrides {
		switch key {
		case "enabled":
			enabled, ok := value.(bool)
			if !ok {
				return parsedSendingQueueOverrides{}, fmt.Errorf("invalid sending_queue.overrides: enabled must be a boolean, got %T", value)
			}
			parsed.enabled = enabled
			parsed.enabledPresent = true
		case "batch":
			batchOverrides, ok := value.(map[string]any)
			if !ok {
				return parsedSendingQueueOverrides{}, fmt.Errorf("invalid sending_queue.overrides: batch must be a map, got %T", value)
			}
			parsed.batchPresent = true
			parsed.batchEnabled = true
			parsed.batchOverrides = make(map[string]any, len(batchOverrides))
			for batchKey, batchValue := range batchOverrides {
				if batchKey != "enabled" {
					parsed.batchOverrides[batchKey] = batchValue
					continue
				}
				batchEnabled, ok := batchValue.(bool)
				if !ok {
					return parsedSendingQueueOverrides{}, fmt.Errorf("invalid sending_queue.overrides.batch: enabled must be a boolean, got %T", batchValue)
				}
				parsed.batchEnabled = batchEnabled
				parsed.batchEnabledPresent = true
			}
		default:
			parsed.queue[key] = value
		}
	}
	return parsed, nil
}

func sizerExpression(value string) (string, error) {
	switch value {
	case "bytes":
		return "exporterhelper.RequestSizerTypeBytes", nil
	case "items":
		return "exporterhelper.RequestSizerTypeItems", nil
	case "requests":
		return "exporterhelper.RequestSizerTypeRequests", nil
	default:
		return "", fmt.Errorf("invalid sending_queue sizer %q", value)
	}
}
