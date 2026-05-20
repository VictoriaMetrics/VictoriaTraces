package tempo

import (
	"testing"
)

// TestTempoBucketNs covers the Log2Bucketize re-binning used to match Tempo's
// histogram_over_time response shape.
func TestTempoBucketNs(t *testing.T) {
	f := func(vmrange string, want uint64) {
		t.Helper()
		got := tempoBucketNs(vmrange)
		if got != want {
			t.Fatalf("tempoBucketNs(%q) = %d; want %d", vmrange, got, want)
		}
	}

	// Exact power-of-2 upper bounds round to themselves.
	f("0...8192", 8192)
	f("0...16384", 16384)
	f("4096...8192", 8192)

	// Non-power-of-2 upper bounds round up to the next power of 2.
	f("0...100000", 131072)      // ceil(log2(100000))=17 → 2^17 = 131072
	f("50000...106583", 131072)  // upper 106583 → 2^17
	f("106583...226974", 262144) // upper 226974 → 2^18

	// Realistic vmrange in scientific notation (nanosecond boundaries).
	f("5.995e+08...6.813e+08", 1<<30) // hi=681300000 → 2^30 = 1073741824

	// Single-value vmrange — VictoriaLogs emits "0" or a single number for
	// degenerate buckets.
	f("0", 0)
	f("1", 1)
	f("8192", 8192)
	f("8193", 16384)

	// Sub-nanosecond underflow buckets (VL emits "0...1.000e-09" for
	// zero-duration spans) must return 0 so the caller drops them — Tempo
	// never emits such a bucket.
	f("0...1.000e-09", 0)
	f("0...0.5", 0)

	// Malformed input must not panic; falls through to 0.
	f("garbage", 0)
}
