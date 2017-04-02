package storage

import (
	"sort"
	"time"

	"github.com/weaveworks-experiments/loki/pkg/model"
)

func mergeStringLists(a, b []string) []string {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	i, j := 0, 0
	var result []string
	for i < len(a) && j < len(b) {
		if a[i] < b[j] {
			result = append(result, a[i])
			i++
		} else if a[i] > b[j] {
			result = append(result, b[j])
			j++
		} else {
			result = append(result, a[i])
			i++
			j++
		}
	}
	for i < len(a) {
		result = append(result, a[i])
		i++
	}
	for j < len(b) {
		result = append(result, b[j])
		j++
	}
	return result
}

func mergeStringListList(ss [][]string) []string {
	switch len(ss) {
	case 0:
		return nil
	case 1:
		return ss[0]
	case 2:
		return mergeStringLists(ss[0], ss[1])
	default:
		midpoint := len(ss) / 2
		return mergeStringLists(mergeStringListList(ss[:midpoint]), mergeStringListList(ss[midpoint:]))
	}
}

// mergeTraceList merges a list of traces into a single trace.  They must all
// have the same traceID.
func mergeTraceList(input []Trace) Trace {
	if len(input) == 0 {
		panic("Cannot merge zero-length list!")
	}

	spans := []*model.Span{}
	var minTimestamp, maxTimestamp time.Time
	for _, trace := range input {
		spans = append(spans, trace.Spans...)
		if !minTimestamp.IsZero() || minTimestamp.After(trace.MinTimestamp) {
			minTimestamp = trace.MinTimestamp
		}
		if maxTimestamp.IsZero() || maxTimestamp.Before(trace.MaxTimestamp) {
			maxTimestamp = trace.MaxTimestamp
		}
	}

	return Trace{
		ID:           input[0].ID,
		MinTimestamp: minTimestamp,
		MaxTimestamp: maxTimestamp,
		Spans:        spans,
	}
}

// mergeTraceListList merges a list of lists traces.  It assumes traces within
// each inner-list do not overlap.
func mergeTraceListList(input [][]Trace) []Trace {
	traces := map[uint64][]Trace{}
	for _, traceList := range input {
		for _, trace := range traceList {
			id := trace.ID
			traces[id] = append(traces[id], trace)
		}
	}

	result := []Trace{}
	for _, traceList := range traces {
		result = append(result, mergeTraceList(traceList))
	}

	sort.Sort(ByMinTimestamp(result))
	return result
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
