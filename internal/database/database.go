package database

import (
	"github.com/mrowling/carelink-go/internal/types"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
}

// New creates a new database connection and initializes tables
func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{conn: conn}

	// Initialize tables
	if err := db.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// initTables creates the database tables if they don't exist
func (db *DB) initTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS glucose_entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL,
		sgv INTEGER NOT NULL,
		sgv_mmol REAL NOT NULL,
		date INTEGER NOT NULL UNIQUE,
		date_string TEXT NOT NULL,
		device TEXT NOT NULL,
		direction TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_glucose_date ON glucose_entries(date);
	CREATE INDEX IF NOT EXISTS idx_glucose_date_string ON glucose_entries(date_string);

	CREATE TABLE IF NOT EXISTS device_status (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at_timestamp TEXT NOT NULL UNIQUE,
		device TEXT NOT NULL,
		pump_battery_percent INTEGER,
		pump_reservoir REAL,
		pump_iob_bolusiob REAL,
		pump_iob_timestamp TEXT,
		pump_clock TEXT,
		uploader_battery INTEGER,
		connect_sensor_state TEXT,
		connect_calib_status TEXT,
		connect_sensor_duration_hours INTEGER,
		connect_time_to_next_calib_hours INTEGER,
		connect_conduit_in_range INTEGER,
		connect_conduit_medical_device_in_range INTEGER,
		connect_conduit_sensor_in_range INTEGER,
		stored_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_device_status_created_at ON device_status(created_at_timestamp);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// SaveSGVEntries saves glucose entries to the database
func (db *DB) SaveSGVEntries(entries []types.SGVEntry) error {
	if len(entries) == 0 {
		return nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO glucose_entries (type, sgv, sgv_mmol, date, date_string, device, direction)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			sgv = excluded.sgv,
			sgv_mmol = excluded.sgv_mmol,
			direction = excluded.direction
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, entry := range entries {
		_, err := stmt.Exec(
			entry.Type,
			entry.SGV,
			entry.SGVMmol,
			entry.Date,
			entry.DateString,
			entry.Device,
			entry.Direction,
		)
		if err != nil {
			log.Printf("[DB] Warning: failed to insert glucose entry (date=%d): %v", entry.Date, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("[DB] Saved %d glucose entries", len(entries))
	return nil
}

// SaveDeviceStatus saves device status to the database
func (db *DB) SaveDeviceStatus(statuses []types.DeviceStatus) error {
	if len(statuses) == 0 {
		return nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO device_status (
			created_at_timestamp, device,
			pump_battery_percent, pump_reservoir,
			pump_iob_bolusiob, pump_iob_timestamp, pump_clock,
			uploader_battery,
			connect_sensor_state, connect_calib_status,
			connect_sensor_duration_hours, connect_time_to_next_calib_hours,
			connect_conduit_in_range, connect_conduit_medical_device_in_range,
			connect_conduit_sensor_in_range
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(created_at_timestamp) DO UPDATE SET
			pump_battery_percent = excluded.pump_battery_percent,
			pump_reservoir = excluded.pump_reservoir,
			pump_iob_bolusiob = excluded.pump_iob_bolusiob,
			uploader_battery = excluded.uploader_battery
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, status := range statuses {
		var pumpBatteryPercent *int
		if status.Pump.Battery.Percent > 0 {
			pumpBatteryPercent = &status.Pump.Battery.Percent
		}

		var pumpReservoir *float64
		if status.Pump.Reservoir > 0 {
			pumpReservoir = &status.Pump.Reservoir
		}

		var pumpIOBBolusIOB *float64
		if status.Pump.IOB.BolusIOB != nil && *status.Pump.IOB.BolusIOB > 0 {
			pumpIOBBolusIOB = status.Pump.IOB.BolusIOB
		}

		var uploaderBattery *int
		if status.Uploader.Battery > 0 {
			uploaderBattery = &status.Uploader.Battery
		}

		var conduitInRange *int
		if status.Connect.ConduitInRange {
			val := 1
			conduitInRange = &val
		} else {
			val := 0
			conduitInRange = &val
		}

		var conduitMedicalDeviceInRange *int
		if status.Connect.ConduitMedicalDeviceInRange {
			val := 1
			conduitMedicalDeviceInRange = &val
		} else {
			val := 0
			conduitMedicalDeviceInRange = &val
		}

		var conduitSensorInRange *int
		if status.Connect.ConduitSensorInRange {
			val := 1
			conduitSensorInRange = &val
		} else {
			val := 0
			conduitSensorInRange = &val
		}

		_, err := stmt.Exec(
			status.CreatedAt,
			status.Device,
			pumpBatteryPercent,
			pumpReservoir,
			pumpIOBBolusIOB,
			status.Pump.IOB.Timestamp,
			status.Pump.Clock,
			uploaderBattery,
			status.Connect.SensorState,
			status.Connect.CalibStatus,
			status.Connect.SensorDurationHours,
			status.Connect.TimeToNextCalibHours,
			conduitInRange,
			conduitMedicalDeviceInRange,
			conduitSensorInRange,
		)
		if err != nil {
			log.Printf("[DB] Warning: failed to insert device status (created_at=%s): %v", status.CreatedAt, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("[DB] Saved %d device status entries", len(statuses))
	return nil
}

// GetRecentGlucoseEntries retrieves recent glucose entries
func (db *DB) GetRecentGlucoseEntries(limit int) ([]types.SGVEntry, error) {
	query := `
		SELECT type, sgv, sgv_mmol, date, date_string, device, direction
		FROM glucose_entries
		ORDER BY date DESC
		LIMIT ?
	`

	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query glucose entries: %w", err)
	}
	defer rows.Close()

	var entries []types.SGVEntry
	for rows.Next() {
		var entry types.SGVEntry
		var direction sql.NullString

		err := rows.Scan(
			&entry.Type,
			&entry.SGV,
			&entry.SGVMmol,
			&entry.Date,
			&entry.DateString,
			&entry.Device,
			&direction,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if direction.Valid {
			entry.Direction = direction.String
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// GetGlucoseEntriesInRange retrieves glucose entries within a time range
func (db *DB) GetGlucoseEntriesInRange(start, end time.Time) ([]types.SGVEntry, error) {
	startMs := start.UnixMilli()
	endMs := end.UnixMilli()

	query := `
		SELECT type, sgv, sgv_mmol, date, date_string, device, direction
		FROM glucose_entries
		WHERE date >= ? AND date <= ?
		ORDER BY date ASC
	`

	rows, err := db.conn.Query(query, startMs, endMs)
	if err != nil {
		return nil, fmt.Errorf("failed to query glucose entries: %w", err)
	}
	defer rows.Close()

	var entries []types.SGVEntry
	for rows.Next() {
		var entry types.SGVEntry
		var direction sql.NullString

		err := rows.Scan(
			&entry.Type,
			&entry.SGV,
			&entry.SGVMmol,
			&entry.Date,
			&entry.DateString,
			&entry.Device,
			&direction,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if direction.Valid {
			entry.Direction = direction.String
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// GetStats returns statistics about the stored data
func (db *DB) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count total glucose entries
	var glucoseCount int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM glucose_entries").Scan(&glucoseCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count glucose entries: %w", err)
	}
	stats["total_glucose_entries"] = glucoseCount

	// Count device status entries
	var deviceStatusCount int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM device_status").Scan(&deviceStatusCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count device status: %w", err)
	}
	stats["total_device_status_entries"] = deviceStatusCount

	// Get date range
	var minDate, maxDate sql.NullInt64
	err = db.conn.QueryRow("SELECT MIN(date), MAX(date) FROM glucose_entries").Scan(&minDate, &maxDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get date range: %w", err)
	}

	if minDate.Valid && maxDate.Valid {
		stats["earliest_entry"] = time.UnixMilli(minDate.Int64).Format(time.RFC3339)
		stats["latest_entry"] = time.UnixMilli(maxDate.Int64).Format(time.RFC3339)
	}

	return stats, nil
}
