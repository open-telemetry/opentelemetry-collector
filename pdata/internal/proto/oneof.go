package proto

import (
	"math"
	"unsafe"
)

type OneOf struct {
	ptr unsafe.Pointer
	len uint64
}

func (m *OneOf) Int() uint64 {
	return m.len
}

func (m *OneOf) SetInt(val uint64) {
	m.len = val
	m.ptr = nil
}

func (m *OneOf) Float() float64 {
	return math.Float64frombits(m.len)
}

func (m *OneOf) SetFloat(val float64) {
	m.len = math.Float64bits(val)
	m.ptr = nil
}

func (m *OneOf) Bool() bool {
	return m.len != 0
}

func (m *OneOf) SetBool(val bool) {
	if val {
		m.len = 1
	} else {
		m.len = 0
	}
	m.ptr = nil
}

func (m *OneOf) Message() unsafe.Pointer {
	return m.ptr
}

func (m *OneOf) SetMessage(ptr unsafe.Pointer) {
	m.len = 0
	m.ptr = ptr
}

func (m *OneOf) Bytes() *[]byte {
	return (*[]byte)(m.ptr)
}

func (m *OneOf) SetBytes(ptr *[]byte) {
	m.len = 0
	m.ptr = unsafe.Pointer(ptr)
}

func (m *OneOf) String() string {
	return unsafe.String((*byte)(m.ptr), m.len)
}

func (m *OneOf) SetString(str string) {
	m.len = uint64(len(str))
	m.ptr = unsafe.Pointer(unsafe.StringData(str))
}

func (m *OneOf) Reset() {
	*m = OneOf{}
}
