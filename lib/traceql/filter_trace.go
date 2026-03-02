package traceql

// TraceFilter is a filter for span, e.g. `{...}`
type filterTrace struct {
	andFilter filter
}

func (f *filterTrace) String() string {
	if f.andFilter.String() == "" {
		return "*"
	}
	return f.andFilter.String()
}

func (f *filterTrace) GetTraceDurationFilters() []*filterCommon {
	return f.andFilter.GetTraceDurationFilters()
}
