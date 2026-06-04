package testdata

import "testing"

func TestSimple(t *testing.T) {
	result := simple()
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}
