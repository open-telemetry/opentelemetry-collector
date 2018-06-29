package model

import (
	"encoding/json"
	"testing"
)

func TestTraceID(t *testing.T) {
	traceID := TraceID{High: 1, Low: 2}
	if len(traceID.String()) != 32 {
		t.Errorf("Expected zero-padded TraceID to have 32 characters")
	}

	b, err := json.Marshal(traceID)
	if err != nil {
		t.Fatalf("Expected successful json serialization, got error: %+v", err)
	}

	var traceID2 TraceID
	if err = json.Unmarshal(b, &traceID2); err != nil {
		t.Fatalf("Expected successful json deserialization, got error: %+v", err)
	}

	have, err := TraceIDFromHex(traceID.String())
	if err != nil {
		t.Fatalf("Expected traceID got error: %+v", err)
	}
	if traceID.High != have.High || traceID.Low != have.Low {
		t.Errorf("Expected %+v, got %+v", traceID, have)
	}

	traceID = TraceID{High: 0, Low: 2}

	if len(traceID.String()) != 16 {
		t.Errorf("Expected zero-padded TraceID to have 16 characters, got %d", len(traceID.String()))
	}

	have, err = TraceIDFromHex(traceID.String())
	if err != nil {
		t.Fatalf("Expected traceID got error: %+v", err)
	}
	if traceID.High != have.High || traceID.Low != have.Low {
		t.Errorf("Expected %+v, got %+v", traceID, have)
	}

	traceID = TraceID{High: 0, Low: 0}

	if !traceID.Empty() {
		t.Errorf("Expected TraceID to be empty")
	}

	if _, err = TraceIDFromHex("12345678901234zz12345678901234zz"); err == nil {
		t.Errorf("Expected error got nil")
	}

	if err = json.Unmarshal([]byte(`"12345678901234zz12345678901234zz"`), &traceID); err == nil {
		t.Errorf("Expected error got nil")
	}

}
