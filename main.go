package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/mrowling/carelink-go/internal/carelink"
	"github.com/mrowling/carelink-go/internal/config"
	"github.com/mrowling/carelink-go/internal/database"
	"github.com/mrowling/carelink-go/internal/paths"
	"github.com/mrowling/carelink-go/internal/transform"
)

// LatestResponse represents the output format for the latest subcommand
type LatestResponse struct {
	SGVMmol    float64 `json:"sgv_mmol"`
	Direction  string  `json:"direction,omitempty"`
	Trend      *int    `json:"trend,omitempty"`
	AgeMinutes int     `json:"age_minutes"`
}

// ErrorResponse represents an error message in JSON format
type ErrorResponse struct {
	Error string `json:"error"`
}

func main() {
	// Check for subcommands
	if len(os.Args) < 2 {
		printError("No command specified")
		printHelp()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "poll":
		runBridge()
	case "latest":
		runLatest()
	case "help", "--help", "-h":
		printHelp()
	default:
		printError(fmt.Sprintf("Unknown command: %s", os.Args[1]))
		printHelp()
		os.Exit(1)
	}
}

// printHelp displays usage information
func printHelp() {
	fmt.Println(`CareLink Go - Fetch and store glucose data from Medtronic CareLink

Usage:
  carelink-go poll     Run the bridge (fetch and output data continuously)
  carelink-go latest   Output the most recent glucose reading from database
  carelink-go help     Show this help message

Commands:
  poll      Continuously fetch data from CareLink API and output to stdout
            Fetches at configured interval (default: 300 seconds)
            Saves all data to SQLite database

  latest    Query the database for the most recent blood glucose reading
            Output format: {"sgv_mmol": 12.3, "direction": "Flat", "trend": 4, "age_minutes": 5}

Environment Variables:
  CARELINK_DB_PATH    Path to SQLite database (default: ~/.carelink/carelink.db)

Configuration Files:
  .env, my.env        Searched in: current directory, then ~/.carelink/
  logindata.json      Searched in: current directory, then ~/.carelink/
  https.txt           Searched in: current directory, then ~/.carelink/

Examples:
  ./carelink-go poll
  ./carelink-go latest
  ./carelink-go latest | jq '.sgv_mmol'`)
}

// printError outputs an error in JSON format to stderr
func printError(message string) {
	errResp := ErrorResponse{Error: message}
	jsonData, _ := json.MarshalIndent(errResp, "", "  ")
	fmt.Fprintln(os.Stderr, string(jsonData))
}

// runLatest queries the database for the most recent glucose reading
func runLatest() {
	// Get database path from environment or use default
	dbPath := os.Getenv("CARELINK_DB_PATH")
	if dbPath == "" {
		var err error
		dbPath, err = paths.GetDefaultDBPath()
		if err != nil {
			printError(fmt.Sprintf("Failed to get default database path: %v", err))
			os.Exit(1)
		}
	}

	// Check if database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		printError(fmt.Sprintf("Database not found at %s", dbPath))
		os.Exit(1)
	}

	// Open database
	db, err := database.New(dbPath)
	if err != nil {
		printError(fmt.Sprintf("Failed to open database: %v", err))
		os.Exit(1)
	}
	defer db.Close()

	// Get most recent entry
	entries, err := db.GetRecentGlucoseEntries(1)
	if err != nil {
		printError(fmt.Sprintf("Failed to query database: %v", err))
		os.Exit(1)
	}

	if len(entries) == 0 {
		printError("No glucose entries found in database")
		os.Exit(1)
	}

	entry := entries[0]

	// Calculate age in minutes (rounded to nearest whole number)
	entryTime := time.UnixMilli(entry.Date)
	ageMinutes := int(math.Round(time.Since(entryTime).Minutes()))

	// Map direction to trend number
	directionToTrend := map[string]int{
		"NONE":          0,
		"TripleUp":      1,
		"DoubleUp":      1,
		"SingleUp":      2,
		"FortyFiveUp":   3,
		"Flat":          4,
		"FortyFiveDown": 5,
		"SingleDown":    6,
		"DoubleDown":    7,
		"TripleDown":    7,
	}

	// Build response
	response := LatestResponse{
		SGVMmol:    entry.SGVMmol,
		AgeMinutes: ageMinutes,
	}

	// Add direction if available
	if entry.Direction != "" {
		response.Direction = entry.Direction

		// Add trend if direction is mappable
		if trend, ok := directionToTrend[entry.Direction]; ok {
			response.Trend = &trend
		}
	}

	// Output JSON
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		printError(fmt.Sprintf("Failed to marshal JSON: %v", err))
		os.Exit(1)
	}

	fmt.Println(string(jsonData))
}

// runBridge runs the main CareLink bridge loop
func runBridge() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[Bridge] Failed to load config: %v", err)
	}

	// Check if logindata.json exists (will be found via paths.FindFile in auth.go)
	_, err = paths.FindFile("logindata.json")
	if err != nil {
		log.Fatal("[Bridge] No logindata.json found — see README for authentication setup")
	}

	// Create CareLink client
	client, err := carelink.NewClient(cfg)
	if err != nil {
		log.Fatalf("[Bridge] Failed to create client: %v", err)
	}

	// Initialize database
	dbPath := os.Getenv("CARELINK_DB_PATH")
	if dbPath == "" {
		dbPath, err = paths.GetDefaultDBPath()
		if err != nil {
			log.Fatalf("[Bridge] Failed to get default database path: %v", err)
		}
	}
	db, err := database.New(dbPath)
	if err != nil {
		log.Fatalf("[Bridge] Failed to initialize database: %v", err)
	}
	defer db.Close()
	log.Printf("[Bridge] Database initialized at %s", dbPath)

	// Create recency filters
	sgvFilter := transform.NewRecencyFilter()
	deviceStatusFilter := transform.NewRecencyFilter()

	// Start loop
	log.Printf("[Bridge] Starting — interval set to %ds", cfg.Interval)
	log.Println("[Bridge] Mode: stdout-only (not uploading to Nightscout)")
	log.Println("[Bridge] Fetching data now...")

	for {
		// Fetch data from CareLink
		data, err := client.Fetch()
		if err != nil {
			log.Printf("[Bridge] Error: %v", err)
			time.Sleep(time.Duration(cfg.Interval) * time.Second)
			continue
		}

		// Check for empty data
		if data.LastMedicalDeviceDataUpdateServerTime == 0 {
			log.Println("[Bridge] Warning: received empty or invalid data from CareLink")
			if cfg.Verbose {
				jsonData, _ := json.Marshal(data)
				log.Printf("[Bridge] Data: %s", string(jsonData))
			}
			time.Sleep(time.Duration(cfg.Interval) * time.Second)
			continue
		}

		// Transform data
		transformed := transform.Transform(data, cfg.SGVLimit)

		// Filter to get only new entries
		newSGVs := sgvFilter.FilterSGVs(transformed.Entries)
		newDeviceStatus := deviceStatusFilter.FilterDeviceStatus(transformed.DeviceStatus)

		// Save to database
		if err := db.SaveSGVEntries(newSGVs); err != nil {
			log.Printf("[Bridge] Error saving glucose entries to database: %v", err)
		}
		if err := db.SaveDeviceStatus(newDeviceStatus); err != nil {
			log.Printf("[Bridge] Error saving device status to database: %v", err)
		}

		// Output JSON (always output on every fetch as requested)
		output := map[string]interface{}{
			"entries":      newSGVs,
			"devicestatus": newDeviceStatus,
		}

		jsonOutput, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			log.Printf("[Bridge] Error marshaling JSON: %v", err)
		} else {
			fmt.Println(string(jsonOutput))
		}

		if cfg.Verbose {
			nextCheck := time.Now().Add(time.Duration(cfg.Interval) * time.Second)
			log.Printf("[Bridge] Next check in %ds (at %s)", cfg.Interval, nextCheck.Format(time.RFC3339))
		}

		// Sleep until next fetch
		time.Sleep(time.Duration(cfg.Interval) * time.Second)
	}
}
