package insertutil

import (
	"flag"
)

var (
	// MaxFieldsPerLine is the maximum number of fields per line for /insert/* handlers
	MaxFieldsPerLine = flag.Int("insert.maxFieldsPerLine", 1000, "The maximum number of log fields per line, which can be read by /insert/* handlers; "+
		"see https://docs.victoriametrics.com/victorialogs/faq/#how-many-fields-a-single-log-entry-may-contain")
)
