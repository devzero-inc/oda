package resources

import (
	"embed"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"text/template"
	"time"

	"github.com/devzero-inc/oda/collector"
	"github.com/devzero-inc/oda/logging"
	"github.com/devzero-inc/oda/process"
)

// Embedding directory
//
//go:embed views/*
var templateFS embed.FS

func showError(w http.ResponseWriter) {
	tmpl, err := template.ParseFS(templateFS, "views/error.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	loc, _ := time.LoadLocation("Local")
	now := time.Now().In(loc)

	var startMillis, endMillis int64

	start := r.URL.Query().Get("start")
	if start == "" {
		// Default start time to the start of today (00:00:00)
		startTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		startMillis = startTime.UnixMilli()
	} else {
		// Parse the incoming start time and convert it to Unix milliseconds
		if parsedTime, err := time.ParseInLocation("2006-01-02T15:04", start, loc); err == nil {
			startMillis = parsedTime.UnixMilli()
		}
	}

	end := r.URL.Query().Get("end")
	if end == "" {
		// Default end time to the current time
		endMillis = now.UnixMilli()
	} else {
		// Parse the incoming end time and convert it to Unix milliseconds
		if parsedTime, err := time.ParseInLocation("2006-01-02T15:04", end, loc); err == nil {
			endMillis = parsedTime.UnixMilli()
		}
	}

	logging.Log.Debug().Msg("Creating waiting groups")

	// Initialize wait group and channels for concurrent operations
	var wg sync.WaitGroup
	commandsChan := make(chan []*collector.Command, 1)
	processesChan := make(chan []*process.Process, 1)
	timeProcessesChan := make(chan map[int64][]*process.Process, 1)

	logging.Log.Debug().Msg("Fetching data concurrently")

	// Increment wait group count for each concurrent operation
	wg.Add(3)

	// Fetch commands concurrently
	go func() {
		logging.Log.Debug().Msg("Fetching commands")
		defer wg.Done()
		commands, err := collector.GetAllCommandsForPeriod(startMillis, endMillis)
		logging.Log.Debug().Msg("Sending commands")
		if err != nil {
			logging.Log.Err(err).Msg("Failed to fetch commands")
			commandsChan <- nil
			return
		}
		commandsChan <- commands
		logging.Log.Debug().Msg("Fetched commands")
	}()

	// Fetch processes concurrently
	go func() {
		logging.Log.Debug().Msg("Fetching processes")
		defer wg.Done()
		processes, err := process.GetAllProcessesForPeriod(startMillis, endMillis)
		logging.Log.Debug().Msg("Sending processes")
		if err != nil {
			logging.Log.Err(err).Msg("Failed to fetch processes")
			processesChan <- nil
			return
		}
		processesChan <- processes
		logging.Log.Debug().Msg("Fetched processes")
	}()

	// Fetch time processes concurrently
	go func() {
		logging.Log.Debug().Msg("Fetching time processes")
		defer wg.Done()
		timeProcesses, err := process.GetTopProcessesAndMetrics(startMillis, endMillis)
		logging.Log.Debug().Msg("Sending time processes")
		if err != nil {
			logging.Log.Err(err).Msg("Failed to fetch time processes")
			timeProcessesChan <- nil
			return
		}
		timeProcessesChan <- timeProcesses
		logging.Log.Debug().Msg("Fetched time processes")
	}()

	logging.Log.Debug().Msg("Waiting...")

	// Wait for all goroutines to finish
	wg.Wait()
	close(commandsChan)
	close(processesChan)
	close(timeProcessesChan)

	// Receive from channels
	commands := <-commandsChan
	processes := <-processesChan
	timeProcesses := <-timeProcessesChan

	// Check for errors after receiving data
	if commands == nil || processes == nil || timeProcesses == nil {
		showError(w)
		return
	}

	logging.Log.Debug().Msg("Preparing data for rendering")

	commandsJson, err := PrepareCommandCategoriesExecutionTimeChartData(commands)
	if err != nil {
		showError(w)
		return
	}
	processResourceJson, err := PrepareProcessesResourceUsageChartData(processes)
	if err != nil {
		showError(w)
		return
	}
	cpuResourceJson, err := PrepareCPUTimeSeriesChartData(timeProcesses)
	if err != nil {
		showError(w)
		return
	}
	memoryResourceJson, err := PrepareMemoryTimeSeriesChartData(timeProcesses)
	if err != nil {
		showError(w)
		return
	}

	tmpl, err := template.ParseFS(templateFS, "views/index.html")
	if err != nil {
		showError(w)
		return
	}

	logging.Log.Info().Msgf("Rendering template with start: %s, end: %s", start, end)

	if start == "" {
		start = time.UnixMilli(startMillis).UTC().Format("2006-01-02T15:04")
	}

	if end == "" {
		end = time.UnixMilli(endMillis).UTC().Format("2006-01-02T15:04")
	}

	if err := tmpl.Execute(w, map[string]interface{}{
		"CommandsJSON":         commandsJson,
		"ProcessesJSON":        processResourceJson,
		"CPUTimeSeriesJSON":    cpuResourceJson,
		"MemoryTimeSeriesJSON": memoryResourceJson,
		"StartTime":            start,
		"EndTime":              end,
	}); err != nil {
		showError(w)
	}
}

func commandHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	label := queryParams.Get("label")

	if label == "" {
		showError(w)
		return
	}

	loc, _ := time.LoadLocation("Local")
	now := time.Now().In(loc)

	var startMillis, endMillis int64

	start := queryParams.Get("start")
	if start == "" {
		// Default start time to the start of today (00:00:00)
		startTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		startMillis = startTime.UnixMilli()
	} else {
		// Parse the incoming start time and convert it to Unix milliseconds
		if parsedTime, err := time.ParseInLocation("2006-01-02T15:04", start, loc); err == nil {
			startMillis = parsedTime.UnixMilli()
		}
	}

	end := queryParams.Get("end")
	if end == "" {
		// Default end time to the current time
		endMillis = now.UnixMilli()
	} else {
		// Parse the incoming end time and convert it to Unix milliseconds
		if parsedTime, err := time.ParseInLocation("2006-01-02T15:04", end, loc); err == nil {
			endMillis = parsedTime.UnixMilli()
		}
	}

	commands, err := collector.GetAllCommandsForCategoryForPeriod(
		label, startMillis, endMillis)
	if err != nil {
		showError(w)
		return
	}

	commandsJson, err := PrepareCommandsExecutionTimeChartData(commands)
	if err != nil {
		showError(w)
		return
	}

	tmpl, err := template.ParseFS(templateFS, "views/command.html")
	if err != nil {
		showError(w)
		return
	}

	logging.Log.Info().Msgf("Rendering template with start: %s, end: %s", start, end)

	if start == "" {
		start = time.UnixMilli(startMillis).UTC().Format("2006-01-02T15:04")
	}

	if end == "" {
		end = time.UnixMilli(endMillis).UTC().Format("2006-01-02T15:04")
	}

	if err := tmpl.Execute(w, map[string]interface{}{
		"CommandsJSON": commandsJson,
		"StartTime":    start,
		"EndTime":      end,
		"Commands":     commands,
	}); err != nil {
		showError(w)
	}
}

func overviewHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	label := queryParams.Get("id")

	if label == "" {
		showError(w)
		return
	}

	i, err := strconv.ParseInt(label, 10, 64)
	if err != nil {
		showError(w)
		return
	}

	command, err := collector.GetCommandById(i)
	if err != nil {
		showError(w)
		return
	}

	// Initialize wait group and channels for concurrent operations
	var wg sync.WaitGroup
	processesChan := make(chan []*process.Process, 1)
	timeProcessesChan := make(chan map[int64][]*process.Process, 1)

	logging.Log.Debug().Msgf("Start time: %d, End time: %d", command.StartTime, command.EndTime)

	// Increment wait group count for each concurrent operation
	wg.Add(2)

	// Fetch processes concurrently
	go func() {
		logging.Log.Debug().Msg("Fetching overview processes")
		defer wg.Done()
		processes, err := process.GetAllProcessesForPeriod(command.StartTime, command.EndTime)
		logging.Log.Debug().Msg("Sending processes")
		if err != nil {
			logging.Log.Err(err).Msg("Failed to fetch processes")
			processesChan <- nil
			return
		}
		processesChan <- processes
		logging.Log.Debug().Msg("Fetched processes")
	}()

	// Fetch time processes concurrently
	go func() {
		logging.Log.Debug().Msg("Fetching overview time processes")
		defer wg.Done()
		timeProcesses, err := process.GetTopProcessesAndMetrics(command.StartTime, command.EndTime)
		logging.Log.Debug().Msg("Sending time processes")
		if err != nil {
			logging.Log.Err(err).Msg("Failed to fetch time processes")
			timeProcessesChan <- nil
			return
		}
		timeProcessesChan <- timeProcesses
		logging.Log.Debug().Msg("Fetched time processes")
	}()

	logging.Log.Debug().Msg("Waiting...")

	// Wait for all goroutines to finish
	wg.Wait()
	close(processesChan)
	close(timeProcessesChan)

	// Receive from channels
	processes := <-processesChan
	timeProcesses := <-timeProcessesChan

	logging.Log.Debug().Msg("Checking for errors...")

	// Check for errors after receiving data
	if processes == nil || timeProcesses == nil {
		logging.Log.Error().Err(err).Msgf("Failed to fetch processes with length: %d, and time processes with length %d", len(processes), len(timeProcesses))
		showError(w)
		return
	}

	// encode processes in processJson
	processesJson, err := json.Marshal(processes)
	if err != nil {
		logging.Log.Err(err).Msg("Failed to marshal processes")
		showError(w)
		return
	}

	processResourceJson, err := PrepareProcessesResourceUsageChartData(processes)
	if err != nil {
		logging.Log.Err(err).Msg("Failed to prepare process resource usage")
		showError(w)
		return
	}
	cpuResourceJson, err := PrepareCPUTimeSeriesChartData(timeProcesses)
	if err != nil {
		logging.Log.Err(err).Msg("Failed to prepare CPU time series")
		showError(w)
		return
	}
	memoryResourceJson, err := PrepareMemoryTimeSeriesChartData(timeProcesses)
	if err != nil {
		logging.Log.Err(err).Msg("Failed to prepare memory time series")
		showError(w)
		return
	}

	tmpl, err := template.ParseFS(templateFS, "views/overview.html")
	if err != nil {
		logging.Log.Err(err).Msg("Failed to render template")
		showError(w)
		return
	}

	if err := tmpl.Execute(w, map[string]interface{}{
		"ProcessResourceJSON":  processResourceJson,
		"CPUTimeSeriesJSON":    cpuResourceJson,
		"MemoryTimeSeriesJSON": memoryResourceJson,
		"Processes":            processes,
		"ProcessJSON":          string(processesJson),
	}); err != nil {
		logging.Log.Err(err).Msg("Failed to render template")
		showError(w)
	}
}

// Serve registers the HTTP handlers for the application
func Serve() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/command", commandHandler)
	http.HandleFunc("/overview", overviewHandler)
}
