// Code generated from Pkl module `BatchProcessorConfig`. DO NOT EDIT.
package batchprocessor

import (
	"context"

	"github.com/apple/pkl-go/pkl"
)

type BatchProcessorConfig struct {
	Output any `pkl:"output"`
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a BatchProcessorConfig
func LoadFromPath(ctx context.Context, path string) (ret *BatchProcessorConfig, err error) {
	evaluator, err := pkl.NewEvaluator(ctx, pkl.PreconfiguredOptions)
	if err != nil {
		return nil, err
	}
	defer func() {
		cerr := evaluator.Close()
		if err == nil {
			err = cerr
		}
	}()
	ret, err = Load(ctx, evaluator, pkl.FileSource(path))
	return ret, err
}

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a BatchProcessorConfig
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (*BatchProcessorConfig, error) {
	var ret BatchProcessorConfig
	if err := evaluator.EvaluateModule(ctx, source, &ret); err != nil {
		return nil, err
	}
	return &ret, nil
}
