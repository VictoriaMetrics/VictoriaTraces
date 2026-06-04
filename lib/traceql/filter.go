package traceql

// Field is a single field for the log entry.
type Field struct {
	// Name is the name of the field
	Name string

	// Value is the value of the field
	Value string
}

// Reset resets f for future reuse.
func (f *Field) Reset() {
	f.Name = ""
	f.Value = ""
}

// filter must implement filtering for log entries.
type filter interface {
	// String returns string representation of the filter
	String() string

	// updateNeededFields must update pf with fields needed for the filter
	//updateNeededFields(pf *prefixfilter.Filter)

	// matchRow must return true if the current filter matches a row with the given fields
	//matchRow(fields []Field) bool

	// ToLogsQLFilter() logsql

	GetTraceDurationFilters() []*filterCommon

	// GetReferencedFields must return the field names referenced by the filter
	// (in TraceQL form, e.g. "resource.service.name", "kind"). It is used to
	// project only the queried attributes onto matched spans in the search
	// response, mirroring Tempo's behavior.
	GetReferencedFields() []string
}
