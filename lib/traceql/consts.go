package traceql

// maxDictSizeBytes is the maximum length of all the keys in the valuesDict.
//
// Dict is stored in columnsHeader, which is read every time the corresponding block is scanned during search queries.
// So it is better to store bigger values in regular columns in order to speed up search speed.
const maxDictSizeBytes = 256

// maxDictLen is the maximum number of entries in the valuesDict.
//
// it shouldn't exceed 255, since the dict len is marshaled into a single byte.
const maxDictLen = 8
