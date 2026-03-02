package traceql

// filterNoop does nothing
type filterNoop struct {
}

func (fn *filterNoop) String() string {
	return "*"
}

func (fn *filterNoop) GetTraceDurationFilters() []*filterCommon {
	return nil
}
