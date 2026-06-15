package score

import (
	"fmt"
	"math"
	"testing"

	"github.com/padiazg/go-crap/internal/merge"
)

func TestCRAP(t *testing.T) {
	cases := []struct {
		cc   int
		cov  float64
		want float64
	}{
		{1, 100, 1.0},
		{1, 0, 2.0},
		{5, 100, 5.0},
		{5, 0, 30.0},
		{10, 50, 22.5},
		{30, 0, 930.0},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("cc=%d_cov=%.0f", tc.cc, tc.cov), func(t *testing.T) {
			got := CRAP(tc.cc, tc.cov)
			if math.Abs(got-tc.want) > 0.01 {
				t.Errorf("CRAP(%d, %.1f) = %.4f, want %.4f", tc.cc, tc.cov, got, tc.want)
			}
		})
	}
}

func TestScore_Pessimistic(t *testing.T) {
	// Test with missing coverage policy
	cov := 0.0
	entries := []merge.MergedEntry{
		{
			File:       "test.go",
			FuncName:   "TestFunc",
			Complexity: 5,
			Coverage:   &cov,
		},
		{
			File:       "test.go",
			FuncName:   "TestFunc2",
			Complexity: 3,
			Coverage:   nil,
		},
	}
	result := Score(entries, MissingPessimistic)
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	// First entry has coverage = 0
	if result[1].Coverage != 0 {
		t.Errorf("missing coverage should be 0, got %f", result[1].Coverage)
	}
}

func TestScore_Optimistic(t *testing.T) {
	entries := []merge.MergedEntry{
		{
			File:       "test.go",
			FuncName:   "TestFunc",
			Complexity: 3,
			Coverage:   nil,
		},
	}
	result := Score(entries, MissingOptimistic)
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Coverage != 100.0 {
		t.Errorf("optimistic missing should be 100, got %f", result[0].Coverage)
	}
}

func BenchmarkCRAP(b *testing.B) {
	for b.Loop() {
		CRAP(10, 75.0)
	}
}

func TestScore_Skip(t *testing.T) {
	entries := []merge.MergedEntry{
		{
			File:       "test.go",
			FuncName:   "TestFunc",
			Complexity: 3,
			Coverage:   nil,
		},
	}
	result := Score(entries, MissingSkip)
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if !result[0].Skipped {
		t.Error("expected Skipped=true, got false")
	}
}
