package searchutil

import (
	"flag"
	"fmt"
	"net/http"
	"strconv"
)

var allowPartialResponseFlag = flag.Bool("search.allowPartialResponse", false, "Whether to allow returning partial responses when some of vtstorage nodes "+
	"from the -storageNode list are unavailable for querying. This flag works only for cluster setup of VictoriaTraces. "+
	"See https://docs.victoriametrics.com/victorialogs/querying/#partial-responses")

// GetAllowPartialResponse returns the effective allow_partial_response value for r.
func GetAllowPartialResponse(r *http.Request) (bool, error) {
	allowPartialResponse := *allowPartialResponseFlag
	s := r.FormValue("allow_partial_response")
	if s == "" {
		return allowPartialResponse, nil
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return false, fmt.Errorf("cannot parse allow_partial_response=%q as bool: %w", s, err)
	}
	return b, nil
}
