package common

import "errors"

const (
	// OutOfRetentionHeaderName header is for communication between vtstorage and vtselect, to notify
	// vtselect that the query time range is completely out of the retention period.
	OutOfRetentionHeaderName = "VT-Out-Of-Retention"
)

var (
	ErrOutOfRetention = errors.New("request time out of retention")
)
