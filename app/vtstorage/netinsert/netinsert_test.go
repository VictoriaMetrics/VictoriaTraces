package netinsert

import (
	cryptorand "crypto/rand"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/promauth"
	"github.com/cespare/xxhash/v2"
)

func TestStreamRowsTracker(t *testing.T) {
	f := func(rowsCount, streamsCount, nodesCount int) {
		t.Helper()

		// generate stream hashes
		streamHashes := make([]uint64, streamsCount)
		for i := range streamHashes {
			streamHashes[i] = xxhash.Sum64(fmt.Appendf(nil, "stream %d.", i))
		}

		srt := newStreamRowsTracker(nodesCount)

		rng := rand.New(rand.NewSource(0))
		rowsPerNode := make([]uint64, nodesCount)
		for range rowsCount {
			streamIdx := rng.Intn(streamsCount)
			h := streamHashes[streamIdx]
			nodeIdx := srt.getNodeIdx(h, cryptorand.Text())
			rowsPerNode[nodeIdx]++
		}

		// Verify that rows are uniformly distributed among nodes.
		expectedRowsPerNode := float64(rowsCount) / float64(nodesCount)
		for nodeIdx, nodeRows := range rowsPerNode {
			if math.Abs(float64(nodeRows)-expectedRowsPerNode)/expectedRowsPerNode > 0.15 {
				t.Fatalf("non-uniform distribution of rows among nodes; node %d has %d rows, while it must have %v rows; rowsPerNode=%d",
					nodeIdx, nodeRows, expectedRowsPerNode, rowsPerNode)
			}
		}
	}

	rowsCount := 10000
	streamsCount := 9
	nodesCount := 2
	f(rowsCount, streamsCount, nodesCount)

	rowsCount = 10000
	streamsCount = 100
	nodesCount = 2
	f(rowsCount, streamsCount, nodesCount)

	rowsCount = 100000
	streamsCount = 1000
	nodesCount = 9
	f(rowsCount, streamsCount, nodesCount)
}

// TestDoRequestUsesPost checks that the requests to vtstorage go out as POST, with and without
// a request body. vtstorage rejects every other method at the /internal/* endpoints.
//
// See https://github.com/VictoriaMetrics/VictoriaTraces/issues/225
func TestDoRequestUsesPost(t *testing.T) {
	var (
		gotMethods []string
		wg         sync.WaitGroup
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		gotMethods = append(gotMethods, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ac, err := (&promauth.Options{}).NewConfig()
	if err != nil {
		t.Fatalf("cannot create auth config: %s", err)
	}
	addr := strings.TrimPrefix(server.URL, "http://")
	s := NewStorage([]string{addr}, []*promauth.Config{ac}, []bool{false}, 1, true)
	defer s.MustStop()

	sn := s.sns[0]
	// /internal/force_flush carries no body, and it used to go out as GET.
	wg.Add(1)
	if err := sn.doRequest("/internal/force_flush", nil); err != nil {
		t.Fatalf("cannot send the bodyless request: %s", err)
	}
	wg.Add(1)
	if err := sn.doRequest("/internal/insert", strings.NewReader("foobar")); err != nil {
		t.Fatalf("cannot send the request with a body: %s", err)
	}

	wg.Wait()
	want := []string{http.MethodPost, http.MethodPost}
	if !reflect.DeepEqual(gotMethods, want) {
		t.Fatalf("unexpected request methods; got %q; want %q", gotMethods, want)
	}
}
