package tempo

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
)

// Intrinsic span fields have no attribute prefix, so they are not discovered by
// the field_names scan. They must be advertised explicitly under the "intrinsic"
// scope of /api/v2/search/tags, otherwise they never appear in the Grafana
// Traces Drilldown attribute breakdown.
func TestSearchTagsIntrinsicScope(t *testing.T) {
	res, err := searchTags(context.Background(), nil, "{}", "intrinsic", 0, 0, 100)
	if err != nil {
		t.Fatalf("searchTags(scope=intrinsic): %s", err)
	}
	for _, want := range []string{"name", "kind", "status", "duration"} {
		if !slices.Contains(res.intrinsicTagList, want) {
			t.Fatalf("intrinsic scope missing %q; got %v", want, res.intrinsicTagList)
		}
	}
	// scope=intrinsic must not leak into other scopes.
	if len(res.spanTagList) != 0 || len(res.resourceTagList) != 0 {
		t.Fatalf("intrinsic scope must not populate span/resource lists; got span=%v resource=%v",
			res.spanTagList, res.resourceTagList)
	}
}

// The /api/v2/search/tags response must expose an "intrinsic" scope carrying the
// intrinsic tag names, matching Tempo's response shape.
func TestSearchTagsResponseIncludesIntrinsicScope(t *testing.T) {
	out := SearchTagsResponse(nil, nil, nil, nil, nil, []string{"name", "kind", "status", "duration"})

	var parsed struct {
		Scopes []struct {
			Name string   `json:"name"`
			Tags []string `json:"tags"`
		} `json:"scopes"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("response is not valid JSON: %s\n%s", err, out)
	}

	var intrinsic []string
	found := false
	for _, sc := range parsed.Scopes {
		if sc.Name == "intrinsic" {
			found = true
			intrinsic = sc.Tags
		}
	}
	if !found {
		t.Fatalf("response missing intrinsic scope: %s", out)
	}
	for _, want := range []string{"name", "kind", "status", "duration"} {
		if !slices.Contains(intrinsic, want) {
			t.Fatalf("intrinsic scope missing %q; got %v", want, intrinsic)
		}
	}
}
