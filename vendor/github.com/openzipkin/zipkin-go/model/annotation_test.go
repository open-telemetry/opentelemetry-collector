package model

import (
	"encoding/json"
	"testing"
)

func TestAnnotationNegativeTimestamp(t *testing.T) {
	var (
		span SpanModel
		b1   = []byte(`{"annotations":[{"timestamp":-1}]}`)
		b2   = []byte(`{"annotations":[{"timestamp":0}]}`)
	)

	if err := json.Unmarshal(b1, &span); err == nil {
		t.Errorf("Unmarshal should have failed with error, have: %+v", span)
	}

	if err := json.Unmarshal(b2, &span); err == nil {
		t.Errorf("Unmarshal should have failed with error, have: %+v", span)
	}
}
