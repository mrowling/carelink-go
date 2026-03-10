package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/mrowling/carelink-go/internal/paths"
)

type Config struct {
	Username         string
	Password         string
	Server           string
	CountryCode      string
	Language         string
	PatientID        string
	Interval         int
	SGVLimit         int
	MaxRetryDuration int
	Verbose          bool
	UseProxy         bool
}

// Load reads the .env file and environment variables into a Config struct
func Load() (*Config, error) {
	// Try to load .env from multiple locations
	// Priority: 1) current directory, 2) ~/.carelink/
	envFile, err := paths.FindFile(".env")
	if err == nil {
		if err := godotenv.Load(envFile); err == nil {
			log.Printf("[Config] Loaded .env from %s", envFile)
		}
	}

	// Also try my.env (optional override file)
	myEnvFile, err := paths.FindFile("my.env")
	if err == nil {
		if err := godotenv.Load(myEnvFile); err == nil {
			log.Printf("[Config] Loaded my.env from %s", myEnvFile)
		}
	}

	cfg := &Config{
		Username:         getEnv("CARELINK_USERNAME", ""),
		Password:         getEnv("CARELINK_PASSWORD", ""),
		Server:           strings.ToUpper(getEnv("MMCONNECT_SERVER", "EU")),
		CountryCode:      strings.ToLower(getEnv("MMCONNECT_COUNTRYCODE", "gb")),
		Language:         strings.ToLower(getEnv("MMCONNECT_LANGCODE", "en")),
		PatientID:        getEnv("CARELINK_PATIENT", ""),
		Interval:         getEnvInt("CARELINK_INTERVAL", 300),
		SGVLimit:         getEnvInt("CARELINK_SGV_LIMIT", 24),
		MaxRetryDuration: getEnvInt("CARELINK_MAX_RETRY_DURATION", 512),
		Verbose:          !getEnvBool("CARELINK_QUIET", true),
		UseProxy:         getEnvBool("USE_PROXY", true),
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		val = os.Getenv(strings.ToLower(key))
	}
	if val == "" {
		val = os.Getenv("CUSTOMCONNSTR_" + key)
	}
	if val == "" {
		val = os.Getenv("CUSTOMCONNSTR_" + strings.ToLower(key))
	}
	if val == "" {
		return defaultVal
	}
	return val
}

func getEnvInt(key string, defaultVal int) int {
	val := getEnv(key, "")
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}

func getEnvBool(key string, defaultVal bool) bool {
	val := getEnv(key, "")
	if val == "" {
		return defaultVal
	}
	val = strings.ToLower(val)
	if val == "true" || val == "1" || val == "yes" {
		return true
	}
	if val == "false" || val == "0" || val == "no" {
		return false
	}
	return defaultVal
}
