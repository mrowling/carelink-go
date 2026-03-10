package transform

import (
	"github.com/mrowling/carelink-go/internal/types"
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

const staleDataThresholdMinutes = 20

var lastGuess string

// Transform converts CareLink data to output format
func Transform(data *types.CareLinkData, sgvLimit int) *types.TransformResult {
	// Check for stale data
	recency := float64(data.CurrentServerTime-data.LastMedicalDeviceDataUpdateServerTime) / (60 * 1000)
	if recency > staleDataThresholdMinutes {
		log.Printf("[Transform] Stale CareLink data: %.2f minutes old", recency)
		return &types.TransformResult{
			DeviceStatus: []types.DeviceStatus{},
			Entries:      []types.SGVEntry{},
		}
	}

	offset, offsetMs := guessPumpOffset(data)

	entries := sgvEntries(data, offset, offsetMs)
	if sgvLimit > 0 && len(entries) > sgvLimit {
		entries = entries[len(entries)-sgvLimit:]
	}

	return &types.TransformResult{
		DeviceStatus: []types.DeviceStatus{deviceStatusEntry(data, offset, offsetMs)},
		Entries:      entries,
	}
}

func guessPumpOffset(data *types.CareLinkData) (string, int64) {
	// Try parsing as ISO 8601 (RFC3339) with various formats
	var pumpTimeAsIfUTC time.Time
	var err error

	// Try RFC3339Nano first (with milliseconds and timezone)
	pumpTimeAsIfUTC, err = time.Parse(time.RFC3339Nano, data.SMedicalDeviceTime)
	if err != nil {
		// Try RFC3339 (standard ISO 8601)
		pumpTimeAsIfUTC, err = time.Parse(time.RFC3339, data.SMedicalDeviceTime)
	}
	if err != nil {
		// Try without timezone suffix
		pumpTimeAsIfUTC, err = time.Parse("2006-01-02T15:04:05.999Z", data.SMedicalDeviceTime)
	}
	if err != nil {
		log.Printf("[Transform] Failed to parse pump time: %v", err)
		return "+0000", 0
	}

	serverTimeUTC := time.UnixMilli(data.CurrentServerTime)
	hoursFloat := float64(pumpTimeAsIfUTC.Unix()-serverTimeUTC.Unix()) / 3600.0
	hours := int64(math.Round(hoursFloat))

	sign := "+"
	if hours < 0 {
		sign = "-"
	}
	absHours := int64(math.Abs(float64(hours)))

	offset := fmt.Sprintf("%s%02d00", sign, absHours)

	if offset != lastGuess {
		log.Printf("[Transform] Guessed pump timezone %s (pump time: %s; server time: %s)",
			offset, data.SMedicalDeviceTime, serverTimeUTC.Format(time.RFC3339))
		lastGuess = offset
	}

	offsetMs := hours * 60 * 60 * 1000
	return offset, offsetMs
}

func parsePumpTime(pumpTimeString string, offsetMs int64) int64 {
	// Try parsing as ISO 8601 (RFC3339) with various formats
	var t time.Time
	var err error

	// Try RFC3339Nano first (with milliseconds and timezone)
	t, err = time.Parse(time.RFC3339Nano, pumpTimeString)
	if err != nil {
		// Try RFC3339 (standard ISO 8601)
		t, err = time.Parse(time.RFC3339, pumpTimeString)
	}
	if err != nil {
		// Try without timezone suffix
		t, err = time.Parse("2006-01-02T15:04:05.999Z", pumpTimeString)
	}
	if err != nil {
		// Try without milliseconds
		t, err = time.Parse("2006-01-02T15:04:05Z", pumpTimeString)
	}
	if err != nil {
		log.Printf("[Transform] Failed to parse pump time %s: %v", pumpTimeString, err)
		return time.Now().UnixMilli()
	}
	return t.UnixMilli() - offsetMs
}

func timestampAsString(timestamp int64) string {
	if timestamp == 0 {
		return time.Now().UTC().Format(time.RFC3339)
	}
	return time.UnixMilli(timestamp).UTC().Format(time.RFC3339)
}

func deviceName(data *types.CareLinkData) string {
	return "connect-" + strings.ToLower(data.MedicalDeviceFamily)
}

func deviceStatusEntry(data *types.CareLinkData, offset string, offsetMs int64) types.DeviceStatus {
	status := types.DeviceStatus{
		CreatedAt: timestampAsString(data.LastMedicalDeviceDataUpdateServerTime),
		Device:    deviceName(data),
		Uploader: types.UploaderInfo{
			Battery: data.ConduitBatteryLevel,
		},
		Connect: types.ConnectInfo{
			SensorState:                 data.SensorState,
			CalibStatus:                 data.CalibStatus,
			SensorDurationHours:         data.SensorDurationHours,
			TimeToNextCalibHours:        data.TimeToNextCalibHours,
			ConduitInRange:              data.ConduitInRange,
			ConduitMedicalDeviceInRange: data.ConduitMedicalDeviceInRange,
			ConduitSensorInRange:        data.ConduitSensorInRange,
		},
	}

	// For Guardian devices, use medical device battery in uploader
	if data.MedicalDeviceFamily == "GUARDIAN" {
		status.Uploader.Battery = data.MedicalDeviceBatteryLevelPercent
		status.Connect.MedicalDeviceBatteryLevelPercent = &data.MedicalDeviceBatteryLevelPercent
		status.Connect.MedicalDeviceFamily = &data.MedicalDeviceFamily
		return status
	}

	// For pump devices, add pump info
	reservoir := 0.0
	if data.ReservoirRemainingUnits != nil {
		reservoir = *data.ReservoirRemainingUnits
	} else if data.ReservoirAmount != nil {
		reservoir = *data.ReservoirAmount
	}

	iob := types.IOBInfo{
		Timestamp: timestampAsString(data.LastMedicalDeviceDataUpdateServerTime),
	}
	if data.ActiveInsulin != nil && data.ActiveInsulin.Amount >= 0 {
		iob.BolusIOB = &data.ActiveInsulin.Amount
	}

	status.Pump = &types.PumpInfo{
		Battery: types.BatteryInfo{
			Percent: data.MedicalDeviceBatteryLevelPercent,
		},
		Reservoir: reservoir,
		IOB:       iob,
		Clock:     timestampAsString(parsePumpTime(data.SMedicalDeviceTime, offsetMs)),
	}

	return status
}

func sgvEntries(data *types.CareLinkData, offset string, offsetMs int64) []types.SGVEntry {
	if len(data.SGs) == 0 {
		return []types.SGVEntry{}
	}

	entries := []types.SGVEntry{}
	for _, sg := range data.SGs {
		if sg.Kind != "SG" || sg.SG == 0 {
			continue
		}

		timestamp := parsePumpTime(sg.Datetime, offsetMs)
		entry := types.SGVEntry{
			Type:       "sgv",
			SGV:        sg.SG,
			SGVMmol:    MgdlToMmol(sg.SG),
			Date:       timestamp,
			DateString: timestampAsString(timestamp),
			Device:     deviceName(data),
		}
		entries = append(entries, entry)
	}

	// Apply trend data to the most recent SGV
	if len(entries) > 0 && len(data.SGs) > 0 && data.SGs[len(data.SGs)-1].SG != 0 {
		if trendData, ok := CareLinkTrendMap[data.LastSGTrend]; ok {
			lastEntry := &entries[len(entries)-1]
			lastEntry.Direction = trendData.Direction
			lastEntry.Trend = trendData.Trend
		}
	}

	return entries
}
