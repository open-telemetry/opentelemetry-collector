package component

import (
	"fmt"
	"testing"
	"unsafe"
)

func TestStatusEventSize(t *testing.T) {
	fmt.Printf("StatusEvent size=%d", unsafe.Sizeof(StatusEvent{}))
}
