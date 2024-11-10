package process

import (
	"errors"
	"fmt"
	"time"

	"github.com/devzero-inc/oda/database"
	gen "github.com/devzero-inc/oda/gen/api/v1"

	"github.com/rs/zerolog"
)

const (
	PsutilType = "psutil"
	PsType     = "ps"
)

// SystemProcess interface for process collection
type SystemProcess interface {
	Collect() ([]Process, error)
}

// Factory implementation for proc providers
type Factory struct {
	logger zerolog.Logger
}

// NewFactory init Factory implementation for proc providers
func NewFactory(logger zerolog.Logger) *Factory {
	return &Factory{
		logger: logger,
	}
}

// Create creates a new system process collector from the factory
func (f *Factory) Create(pType string) (SystemProcess, error) {
	switch pType {
	case PsutilType:
		return NewPsutil(f.logger), nil
	case PsType:
		return NewPs(f.logger), nil
	default:
		return nil, errors.New("system process type not supported")
	}
}

// Process is the model for process
type Process struct {
	Id   int64  `json:"id" db:"id"`
	PID  int64  `json:"pid" db:"pid"`
	PPID int64  `json:"ppid" db:"ppid"`
	Name string `json:"name" db:"name"`
	// R: Running; S: Sleep; T: Stop; I: Idle; Z: Zombie; W: Wait; L: Lock;
	Status         string  `json:"status" db:"status"`
	CreatedTime    int64   `json:"created_time" db:"created_time"`
	StoredTime     int64   `json:"stored_time" db:"stored_time"`
	OS             string  `json:"os" db:"os"`
	Platform       string  `json:"platform" db:"platform"`
	PlatformFamily string  `json:"platform_family" db:"platform_family"`
	CPUUsage       float64 `json:"cpu_usage" db:"cpu_usage"`
	MemoryUsage    float64 `json:"memory_usage" db:"memory_usage"`
}

// GetAllProcessesForPeriod fetches all processes for a given period
func GetAllProcessesForPeriod(start int64, end int64) ([]*Process, error) {
	var processes []*Process

	query := `SELECT pid, name, MAX(cpu_usage) as cpu_usage, MAX(memory_usage) as memory_usage
FROM (
    SELECT pid, name, cpu_usage, memory_usage
    FROM processes
    WHERE stored_time BETWEEN ? AND ? 
    ORDER BY cpu_usage DESC, memory_usage DESC
    LIMIT 100
) AS filtered_processes
GROUP BY pid, name
ORDER BY cpu_usage DESC, memory_usage DESC;`

	err := database.DB.Select(&processes, query, start, end)
	if err != nil {
		return nil, err
	}

	return processes, nil
}

// GetTopProcessesAndMetrics fetches the top processes based on a criterion like average CPU usage,
// and then fetches detailed time-series data for each top process.
func GetTopProcessesAndMetrics(start int64, end int64) (map[int64][]*Process, error) {
	query := `SELECT p.name, p.pid, p.cpu_usage, p.memory_usage, p.stored_time
FROM (
    SELECT pid, name, MAX(cpu_usage) as cpu_usage, MAX(memory_usage) as memory_usage
        FROM (
            SELECT pid, name, cpu_usage, memory_usage
            FROM processes
            WHERE stored_time BETWEEN ? AND ? 
            ORDER BY cpu_usage DESC, memory_usage DESC
            LIMIT 100
        ) AS filtered_processes
    GROUP BY pid, name
    ORDER BY cpu_usage DESC, memory_usage DESC
    LIMIT 20
) AS top_processes
JOIN processes p ON top_processes.name = p.name AND top_processes.pid = p.pid
WHERE p.stored_time BETWEEN ? AND ? 
ORDER BY p.stored_time DESC;`

	var allMetrics []*Process
	err := database.DB.Select(&allMetrics, query, start, end, start, end)
	if err != nil {
		return nil, fmt.Errorf("error fetching process metrics: %v", err)
	}

	// Organize the results into a map of PID to list of Process structs
	processMetricsMap := make(map[int64][]*Process)
	for _, metric := range allMetrics {
		processMetricsMap[metric.PID] = append(processMetricsMap[metric.PID], metric)
	}

	return processMetricsMap, nil
}

// DeleteProcessesByDays deletes records older than n days
func DeleteProcessesByDays(days int) error {
	// Calculate the time when old records will be deleted
	timeToDelete := time.Now().AddDate(0, 0, -days).Unix()

	result, err := database.DB.Exec("DELETE FROM processes WHERE stored_time < ?", timeToDelete)
	if err != nil {
		return err
	}

	_, err = result.RowsAffected()

	return err
}

// InsertProcesses inserts multiple processes into the database in bulk
func InsertProcesses(processes []Process) error {
	query := `INSERT INTO processes (pid, name, status, created_time, stored_time, os, platform, platform_family, cpu_usage, memory_usage, ppid)
	VALUES (:pid, :name, :status, :created_time, :stored_time, :os, :platform, :platform_family, :cpu_usage, :memory_usage, :ppid)`

	// Begin a transaction
	tx, err := database.DB.Beginx()
	if err != nil {
		return err
	}

	// Prepare the statement for execution, within the transaction
	stmt, err := tx.PrepareNamed(query)
	if err != nil {
		return err
	}
	defer stmt.Close() // Ensure the statement is closed after execution

	for _, process := range processes {
		// Execute the query for each process
		_, err := stmt.Exec(process)
		if err != nil {
			// In case of an error, roll back the transaction
			tx.Rollback()
			return err
		}
	}

	// Commit the transaction after all inserts
	return tx.Commit()
}

func MapProcessToProto(process Process) *gen.Process {
	return &gen.Process{
		Id:             process.Id,
		Pid:            process.PID,
		Ppid:           process.PPID,
		Name:           process.Name,
		Status:         process.Status,
		CreatedTime:    process.CreatedTime,
		StoredTime:     process.StoredTime,
		Os:             process.OS,
		Platform:       process.Platform,
		PlatformFamily: process.PlatformFamily,
		CpuUsage:       process.CPUUsage,
		MemoryUsage:    process.MemoryUsage,
	}
}
