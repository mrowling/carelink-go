package types

// SGVEntry represents a glucose reading in the output format
type SGVEntry struct {
	Type       string  `json:"type"`
	SGV        int     `json:"sgv"`
	SGVMmol    float64 `json:"sgv_mmol"`
	Date       int64   `json:"date"`
	DateString string  `json:"dateString"`
	Device     string  `json:"device"`
	Direction  string  `json:"direction,omitempty"`
	Trend      int     `json:"trend,omitempty"`
}

// UploaderInfo represents uploader battery status
type UploaderInfo struct {
	Battery int `json:"battery"`
}

// BatteryInfo represents pump battery information
type BatteryInfo struct {
	Percent int `json:"percent"`
}

// IOBInfo represents insulin on board information
type IOBInfo struct {
	Timestamp string   `json:"timestamp"`
	BolusIOB  *float64 `json:"bolusiob,omitempty"`
}

// PumpInfo represents pump status information
type PumpInfo struct {
	Battery   BatteryInfo `json:"battery"`
	Reservoir float64     `json:"reservoir"`
	IOB       IOBInfo     `json:"iob"`
	Clock     string      `json:"clock"`
}

// ConnectInfo represents CareLink Connect status
type ConnectInfo struct {
	SensorState                      string  `json:"sensorState"`
	CalibStatus                      string  `json:"calibStatus"`
	SensorDurationHours              int     `json:"sensorDurationHours"`
	TimeToNextCalibHours             int     `json:"timeToNextCalibHours"`
	ConduitInRange                   bool    `json:"conduitInRange"`
	ConduitMedicalDeviceInRange      bool    `json:"conduitMedicalDeviceInRange"`
	ConduitSensorInRange             bool    `json:"conduitSensorInRange"`
	MedicalDeviceBatteryLevelPercent *int    `json:"medicalDeviceBatteryLevelPercent,omitempty"`
	MedicalDeviceFamily              *string `json:"medicalDeviceFamily,omitempty"`
}

// DeviceStatus represents device status information
type DeviceStatus struct {
	CreatedAt string       `json:"created_at"`
	Device    string       `json:"device"`
	Uploader  UploaderInfo `json:"uploader"`
	Pump      *PumpInfo    `json:"pump,omitempty"`
	Connect   ConnectInfo  `json:"connect"`
}

// TransformResult is the final output structure
type TransformResult struct {
	DeviceStatus []DeviceStatus `json:"devicestatus"`
	Entries      []SGVEntry     `json:"entries"`
}
