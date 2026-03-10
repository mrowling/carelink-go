package types

// CareLinkSG represents a single sensor glucose reading
type CareLinkSG struct {
	SG         int    `json:"sg"`
	Datetime   string `json:"datetime"`
	Version    int    `json:"version"`
	TimeChange bool   `json:"timeChange"`
	Kind       string `json:"kind"`
}

// CareLinkActiveInsulin represents insulin on board data
type CareLinkActiveInsulin struct {
	Datetime string  `json:"datetime"`
	Version  int     `json:"version"`
	Amount   float64 `json:"amount"`
	Kind     string  `json:"kind"`
}

// CareLinkAlarm represents a pump alarm
type CareLinkAlarm struct {
	Type     string `json:"type"`
	Version  int    `json:"version"`
	Flash    bool   `json:"flash"`
	Datetime string `json:"datetime"`
	Kind     string `json:"kind"`
	Code     int    `json:"code"`
}

// CareLinkData is the main response structure from CareLink API
type CareLinkData struct {
	SGs                                   []CareLinkSG           `json:"sgs"`
	LastSG                                CareLinkSG             `json:"lastSG"`
	LastSGTrend                           string                 `json:"lastSGTrend"`
	CurrentServerTime                     int64                  `json:"currentServerTime"`
	SMedicalDeviceTime                    string                 `json:"sMedicalDeviceTime"`
	LastMedicalDeviceDataUpdateServerTime int64                  `json:"lastMedicalDeviceDataUpdateServerTime"`
	MedicalDeviceFamily                   string                 `json:"medicalDeviceFamily"`
	MedicalDeviceBatteryLevelPercent      int                    `json:"medicalDeviceBatteryLevelPercent"`
	ConduitBatteryLevel                   int                    `json:"conduitBatteryLevel"`
	ConduitBatteryStatus                  string                 `json:"conduitBatteryStatus"`
	ConduitInRange                        bool                   `json:"conduitInRange"`
	ConduitMedicalDeviceInRange           bool                   `json:"conduitMedicalDeviceInRange"`
	ConduitSensorInRange                  bool                   `json:"conduitSensorInRange"`
	SensorState                           string                 `json:"sensorState"`
	CalibStatus                           string                 `json:"calibStatus"`
	SensorDurationHours                   int                    `json:"sensorDurationHours"`
	TimeToNextCalibHours                  int                    `json:"timeToNextCalibHours"`
	ReservoirRemainingUnits               *float64               `json:"reservoirRemainingUnits,omitempty"`
	ReservoirAmount                       *float64               `json:"reservoirAmount,omitempty"`
	ActiveInsulin                         *CareLinkActiveInsulin `json:"activeInsulin,omitempty"`
	LastAlarm                             *CareLinkAlarm         `json:"lastAlarm,omitempty"`
	BGUnits                               string                 `json:"bgUnits,omitempty"`
	BGUnitsLower                          string                 `json:"bgunits,omitempty"`
	TimeFormat                            string                 `json:"timeFormat,omitempty"`
}

// UserInfo represents CareLink user information
type UserInfo struct {
	ID       string `json:"id,omitempty"`
	Username string `json:"username,omitempty"`
	Role     string `json:"role"`
}

// PatientLink represents a linked patient for care partner accounts
type PatientLink struct {
	Username string `json:"username"`
}

// CountrySettings contains country-specific API endpoints
type CountrySettings struct {
	BLEPeriodicDataEndpoint string `json:"blePereodicDataEndpoint,omitempty"`
}

// LoginData represents stored OAuth2 tokens
type LoginData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ClientID     string `json:"client_id"`
	TokenURL     string `json:"token_url"`
	Audience     string `json:"audience,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// TokenResponse is the OAuth2 token refresh response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}
