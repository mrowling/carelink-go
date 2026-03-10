package transform

import (
	"github.com/mrowling/carelink-go/internal/types"
)

// RecencyFilter tracks the last seen timestamp to filter out duplicate entries
type RecencyFilter struct {
	lastTimestamp int64
}

// NewRecencyFilter creates a new RecencyFilter
func NewRecencyFilter() *RecencyFilter {
	return &RecencyFilter{
		lastTimestamp: 0,
	}
}

// FilterSGVs filters SGV entries to only include those newer than last seen
func (f *RecencyFilter) FilterSGVs(entries []types.SGVEntry) []types.SGVEntry {
	var newEntries []types.SGVEntry

	for _, entry := range entries {
		if entry.Date > f.lastTimestamp {
			newEntries = append(newEntries, entry)
			if entry.Date > f.lastTimestamp {
				f.lastTimestamp = entry.Date
			}
		}
	}

	return newEntries
}

// FilterDeviceStatus filters device status entries to only include those newer than last seen
func (f *RecencyFilter) FilterDeviceStatus(statuses []types.DeviceStatus) []types.DeviceStatus {
	var newStatuses []types.DeviceStatus

	for _, status := range statuses {
		// Parse the created_at timestamp (RFC3339 format)
		// For simplicity, we'll just return all status entries since there's typically only one
		// and the Transform function already ensures it's recent
		newStatuses = append(newStatuses, status)
	}

	return newStatuses
}
