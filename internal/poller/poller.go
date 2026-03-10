package poller

import (
	"encoding/json"
	"time"

	"github.com/mrowling/carelink-go/internal/carelink"
	"github.com/mrowling/carelink-go/internal/config"
	"github.com/mrowling/carelink-go/internal/database"
	"github.com/mrowling/carelink-go/internal/logger"
	"github.com/mrowling/carelink-go/internal/transform"
)

type Poller struct {
	client  *carelink.Client
	db      *database.DB
	config  *config.Config
	onFetch func(time.Time) // Callback to update server's lastFetch
}

func New(client *carelink.Client, db *database.DB, cfg *config.Config, onFetch func(time.Time)) *Poller {
	return &Poller{
		client:  client,
		db:      db,
		config:  cfg,
		onFetch: onFetch,
	}
}

func (p *Poller) Start() {
	logger.Info("Poller", "Starting with interval %ds", p.config.Interval)

	ticker := time.NewTicker(time.Duration(p.config.Interval) * time.Second)
	defer ticker.Stop()

	// Fetch immediately on start
	p.fetch()

	// Then fetch on interval
	for range ticker.C {
		p.fetch()
	}
}

func (p *Poller) fetch() {
	logger.Info("Poller", "Fetching data from CareLink API")

	result, err := p.client.Fetch()
	if err != nil {
		logger.Error("Poller", "Fetch failed: %v", err)
		return
	}

	if result == nil || len(result.SGs) == 0 {
		logger.Warn("Poller", "Received empty or invalid data from CareLink")
		return
	}

	// Log full JSON at DEBUG level
	if jsonData, err := json.Marshal(result); err == nil {
		logger.Debug("Poller", "Raw API response: %s", string(jsonData))
	}

	// Transform data
	transformed := transform.Transform(result, p.config.SGVLimit)

	// Save to database
	if len(transformed.Entries) > 0 {
		if err := p.db.SaveSGVEntries(transformed.Entries); err != nil {
			logger.Error("Poller", "Failed to save glucose entries: %v", err)
		}
	}

	if len(transformed.DeviceStatus) > 0 {
		if err := p.db.SaveDeviceStatus(transformed.DeviceStatus); err != nil {
			logger.Error("Poller", "Failed to save device status: %v", err)
		}
	}

	// Update last fetch time
	if p.onFetch != nil {
		p.onFetch(time.Now())
	}

	logger.Info("Poller", "Data fetched successfully: %d glucose entries, %d device status",
		len(transformed.Entries), len(transformed.DeviceStatus))
}
