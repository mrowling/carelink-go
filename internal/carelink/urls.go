package carelink

import "fmt"

const (
	defaultServerEU = "carelink.minimed.eu"
	defaultServerUS = "carelink.minimed.com"
)

// URLs holds all CareLink API endpoints
type URLs struct {
	Me              string
	CountrySettings string
	MonitorData     string
	LinkedPatients  string
	serverName      string
}

// ResolveServerName determines the server hostname based on region
func ResolveServerName(server, serverName string) string {
	if serverName != "" {
		return serverName
	}
	if server == "EU" {
		return defaultServerEU
	}
	if server == "US" {
		return defaultServerUS
	}
	if server != "" {
		return server
	}
	return defaultServerEU
}

// NewURLs creates a URLs struct with all endpoints configured
func NewURLs(serverName, countryCode, lang string) *URLs {
	return &URLs{
		Me:              fmt.Sprintf("https://%s/patient/users/me", serverName),
		CountrySettings: fmt.Sprintf("https://%s/patient/countries/settings?countryCode=%s&language=%s", serverName, countryCode, lang),
		MonitorData:     fmt.Sprintf("https://%s/patient/monitor/data", serverName),
		LinkedPatients:  fmt.Sprintf("https://%s/patient/m2m/links/patients", serverName),
		serverName:      serverName,
	}
}

// ConnectData returns the connect data endpoint with timestamp
func (u *URLs) ConnectData(timestamp int64) string {
	return fmt.Sprintf("https://%s/patient/connect/data?cpSerialNumber=NONE&msgType=last24hours&requestTime=%d",
		u.serverName, timestamp)
}
