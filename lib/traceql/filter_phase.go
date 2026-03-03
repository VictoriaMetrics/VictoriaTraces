package traceql

import "sync"

type filterPhrase struct {
	fieldName string
	phrase    string

	tokensOnce   sync.Once
	tokens       []string
	tokensHashes []uint64
}

func (fp *filterPhrase) String() string {
	return quoteFieldNameIfNeeded(fp.fieldName) + "=" + quoteTokenIfNeeded(fp.phrase)
}

func (fp *filterPhrase) GetTraceDurationFilters() []*filterCommon {
	return nil
}
