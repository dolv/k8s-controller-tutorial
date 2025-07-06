package testutil

import (
	"github.com/dolv/k8s-controller-tutorial/internal/utils"
	"testing"
)

func TestInt32Ptr(t *testing.T) {
	v := int32(42)
	ptr := utils.Int32Ptr(v)
	if ptr == nil || *ptr != v {
		t.Errorf("Int32Ptr(%d) = %v, want pointer to %d", v, ptr, v)
	}
}
