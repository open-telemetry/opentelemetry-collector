package pointers

import (
	"testing"
	"time"
)

func TestToString(t *testing.T) {
	type testCase struct {
		caseName string
		ptrValue *string
		defValue string
		expected string
	}

	testCases := []testCase{
		{
			"from nil pointer",
			nil,
			"default value",
			"default value",
		},
		{
			"from pointer with empty value",
			FromString(""),
			"default value",
			"",
		},
		{
			"from pointer with non empty value",
			FromString("value"),
			"deafult value",
			"value",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.caseName, func(t *testing.T) {
			if actual := ToString(tc.ptrValue, tc.defValue); actual != tc.expected {
				t.Error("actual value invalid:", actual, "expected:", tc.expected)
			}
		})
	}
}

func TestToInt(t *testing.T) {
	type testCase struct {
		caseName string
		ptrValue *int
		defValue int
		expected int
	}

	testCases := []testCase{
		{
			"from nil pointer",
			nil,
			17,
			17,
		},
		{
			"from pointer with zero int",
			FromInt(0),
			18,
			0,
		},
		{
			"from pointer with non zero value",
			FromInt(19),
			21,
			19,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.caseName, func(t *testing.T) {
			if actual := ToInt(tc.ptrValue, tc.defValue); actual != tc.expected {
				t.Error("actual value invalid:", actual, "expected:", tc.expected)
			}
		})
	}
}

func TestToInt32(t *testing.T) {
	type testCase struct {
		caseName string
		ptrValue *int32
		defValue int32
		expected int32
	}

	testCases := []testCase{
		{
			"from nil pointer",
			nil,
			17,
			17,
		},
		{
			"from pointer with zero int",
			FromInt32(0),
			18,
			0,
		},
		{
			"from pointer with non zero value",
			FromInt32(19),
			21,
			19,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.caseName, func(t *testing.T) {
			if actual := ToInt32(tc.ptrValue, tc.defValue); actual != tc.expected {
				t.Error("actual value invalid:", actual, "expected:", tc.expected)
			}
		})
	}
}

func TestToInt64(t *testing.T) {
	type testCase struct {
		caseName string
		ptrValue *int64
		defValue int64
		expected int64
	}

	testCases := []testCase{
		{
			"from nil pointer",
			nil,
			17,
			17,
		},
		{
			"from pointer with zero int",
			FromInt64(0),
			18,
			0,
		},
		{
			"from pointer with non zero value",
			FromInt64(19),
			21,
			19,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.caseName, func(t *testing.T) {
			if actual := ToInt64(tc.ptrValue, tc.defValue); actual != tc.expected {
				t.Error("actual value invalid:", actual, "expected:", tc.expected)
			}
		})
	}
}

func TestToBool(t *testing.T) {
	type testCase struct {
		caseName string
		ptrValue *bool
		defValue bool
		expected bool
	}

	testCases := []testCase{
		{
			"from nil pointer true",
			nil,
			true,
			true,
		},
		{
			"from nil pointer false",
			nil,
			false,
			false,
		},
		{
			"from pointer with false",
			FromBool(false),
			true,
			false,
		},
		{
			"from pointer with true",
			FromBool(true),
			false,
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.caseName, func(t *testing.T) {
			if actual := ToBool(tc.ptrValue, tc.defValue); actual != tc.expected {
				t.Error("actual value invalid:", actual, "expected:", tc.expected)
			}
		})
	}
}

func TestToTime(t *testing.T) {
	type testCase struct {
		caseName string
		ptrValue *time.Time
		defValue time.Time
		expected time.Time
	}

	now := time.Now()

	testCases := []testCase{
		{
			"from nil pointer",
			nil,
			now,
			now,
		},
		{
			"from pointer with empty time",
			FromTime(time.Time{}),
			now,
			time.Time{},
		},
		{
			"from pointer with non zero value",
			FromTime(now),
			time.Now().Add(5 * 3600),
			now,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.caseName, func(t *testing.T) {
			if actual := ToTime(tc.ptrValue, tc.defValue); actual != tc.expected {
				t.Error("actual value invalid:", actual, "expected:", tc.expected)
			}
		})
	}
}

func TestToFloat32(t *testing.T) {
	type testCase struct {
		caseName string
		ptrValue *float32
		defValue float32
		expected float32
	}

	testCases := []testCase{
		{
			"from nil pointer",
			nil,
			17,
			17,
		},
		{
			"from pointer with zero value",
			FromFloat32(0),
			18,
			0,
		},
		{
			"from pointer with non zero value",
			FromFloat32(19),
			21,
			19,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.caseName, func(t *testing.T) {
			if actual := ToFloat32(tc.ptrValue, tc.defValue); actual != tc.expected {
				t.Error("actual value invalid:", actual, "expected:", tc.expected)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	type testCase struct {
		caseName string
		ptrValue *float64
		defValue float64
		expected float64
	}

	testCases := []testCase{
		{
			"from nil pointer",
			nil,
			17,
			17,
		},
		{
			"from pointer with zero value",
			FromFloat64(0),
			18,
			0,
		},
		{
			"from pointer with non zero value",
			FromFloat64(19),
			21,
			19,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.caseName, func(t *testing.T) {
			if actual := ToFloat64(tc.ptrValue, tc.defValue); actual != tc.expected {
				t.Error("actual value invalid:", actual, "expected:", tc.expected)
			}
		})
	}
}
