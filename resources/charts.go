package resources

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/devzero-inc/oda/collector"
	"github.com/devzero-inc/oda/process"
)

// ChartData represents the overall structure for a Chart.js chart configuration.
type ChartData struct {
	Type    string        `json:"type"`
	Data    ChartDataData `json:"data"`
	Options ChartOptions  `json:"options"`
}

// ChartDataData holds the datasets and labels for the chart.
type ChartDataData struct {
	Ids      []int              `json:"ids"`
	Datasets []ChartDataDataset `json:"datasets"`
	Labels   []string           `json:"labels,omitempty"`
}

// ChartDataDataset represents a dataset within a Chart.js chart, including its data points and styling.
type ChartDataDataset struct {
	Label           string      `json:"label"`
	Data            interface{} `json:"data"`
	BackgroundColor string      `json:"backgroundColor,omitempty"`
	BorderColor     string      `json:"borderColor,omitempty"`
	Fill            bool        `json:"fill"`
	Tension         float64     `json:"tension,omitempty"`
	PointStyle      string      `json:"pointStyle,omitempty"`
	PointRadius     interface{} `json:"pointRadius,omitempty"` // Can be a static value or an array
	BorderWidth     int         `json:"borderWidth,omitempty"`
}

// DataPoint represents an individual data point in the dataset, used for charts that plot points on axes.
type DataPoint struct {
	X           interface{} `json:"x"` // Can be a string or numeric type, depending on the chart type
	Y           interface{} `json:"y"` // Can be a string or numeric type, depending on the chart type
	ProcessName string      `json:"processName,omitempty"`
	R           float64     `json:"r,omitempty"`
}

// ChartOptions encapsulates all configuration options for the chart, such as scales and plugins.
type ChartOptions struct {
	Scales              *ChartScales  `json:"scales,omitempty"`
	Plugins             *ChartPlugins `json:"plugins,omitempty"`
	MaintainAspectRatio bool          `json:"maintainAspectRatio"`
	Responsive          bool          `json:"responsive"`
	AspectRatio         int           `json:"aspectRatio,omitempty"`
}

// ChartScales defines the axes of the chart, including their types and specific configurations.
type ChartScales struct {
	XAxes ChartAxisOptions `json:"x,omitempty"`
	YAxes ChartAxisOptions `json:"y,omitempty"`
}

// ChartAxisOptions represents configuration options for a single axis in the chart.
type ChartAxisOptions struct {
	Type        string            `json:"type,omitempty"`
	Position    string            `json:"position,omitempty"`
	Time        *ChartTimeOptions `json:"time,omitempty"`
	Title       *ChartAxisTitle   `json:"title,omitempty"`
	BeginAtZero bool              `json:"beginAtZero"`
}

// ChartTimeOptions is used for axes that represent time, allowing for the specification of time units and formats.
type ChartTimeOptions struct {
	Unit string `json:"unit,omitempty"`
}

// ChartAxisTitle defines the title for an axis, including whether it is displayed.
type ChartAxisTitle struct {
	Display bool   `json:"display"`
	Text    string `json:"text"`
}

// ChartPlugins allows for the configuration of various Chart.js plugins, such as legends and tooltips.
type ChartPlugins struct {
	Legend  *ChartLegendOptions  `json:"legend,omitempty"`
	Tooltip *ChartTooltipOptions `json:"tooltip,omitempty"`
}

// ChartLegendOptions configures the chart's legend.
type ChartLegendOptions struct {
	Display  bool                `json:"display"`
	Position string              `json:"position,omitempty"`
	Align    string              `json:"align,omitempty"`
	Label    *LegendLabelOptions `json:"labels,omitempty"`
}

// LegendLabelOptions configures the chart's legend label.
type LegendLabelOptions struct {
	BoxWidth  int    `json:"boxWidth"`
	BoxHeight int    `json:"boxHeight"`
	Color     string `json:"color"`
	Padding   int    `json:"padding"`
}

// ChartTooltipOptions configures the chart's tooltips.
type ChartTooltipOptions struct {
	Enabled bool `json:"enabled"`
}

// PrepareCPUTimeSeriesChartData prepares the data for the CPU Time Series chart.
func PrepareCPUTimeSeriesChartData(processData map[int64][]*process.Process) (string, error) {

	if (processData == nil) || (len(processData) == 0) {
		return "", nil
	}

	var datasets []ChartDataDataset
	for pid, processes := range processData {
		var dataPoints []DataPoint
		for _, proc := range processes {
			dataPoints = append(dataPoints, DataPoint{
				X: proc.StoredTime,
				Y: proc.CPUUsage,
			})
		}

		datasets = append(datasets, ChartDataDataset{
			Label:   fmt.Sprintf("%s - %d", processes[0].Name, pid),
			Data:    dataPoints,
			Fill:    false,
			Tension: 0.1,
		})
	}

	// Complete chart data with options
	chartData := ChartData{
		Type: "line",
		Data: ChartDataData{
			Datasets: datasets,
		},
		Options: ChartOptions{
			Scales: &ChartScales{
				XAxes: ChartAxisOptions{
					Type:     "linear",
					Position: "bottom",
					Title: &ChartAxisTitle{
						Display: true,
						Text:    "Time",
					},
				},
				YAxes: ChartAxisOptions{
					BeginAtZero: true,
					Title: &ChartAxisTitle{
						Display: true,
						Text:    "CPU Usage (%)",
					},
				},
			},
			Plugins: &ChartPlugins{
				Legend: &ChartLegendOptions{
					Display: false,
				},
			},
			MaintainAspectRatio: false,
			Responsive:          true,
		},
	}

	chartJSON, err := json.Marshal(chartData)
	if err != nil {
		return "", err
	}

	return string(chartJSON), nil
}

// PrepareMemoryTimeSeriesChartData prepares and returns the chart data for memory usage as a JSON string.
func PrepareMemoryTimeSeriesChartData(processData map[int64][]*process.Process) (string, error) {

	if (processData == nil) || (len(processData) == 0) {
		return "", nil
	}

	var datasets []ChartDataDataset
	for pid, processes := range processData {
		var dataPoints []DataPoint
		for _, proc := range processes {
			dataPoints = append(dataPoints, DataPoint{
				X: proc.StoredTime,
				Y: proc.MemoryUsage,
			})
		}

		datasets = append(datasets, ChartDataDataset{
			Label:   fmt.Sprintf("%s - PID: %d", processes[0].Name, pid),
			Data:    dataPoints,
			Fill:    false,
			Tension: 0.1,
		})
	}

	chartData := ChartData{
		Type: "line",
		Data: ChartDataData{
			Datasets: datasets,
		},
		Options: ChartOptions{
			Scales: &ChartScales{
				XAxes: ChartAxisOptions{
					Type:     "linear",
					Position: "bottom",
					Title: &ChartAxisTitle{
						Display: true,
						Text:    "Time",
					},
				},
				YAxes: ChartAxisOptions{
					BeginAtZero: true,
					Title: &ChartAxisTitle{
						Display: true,
						Text:    "Memory Usage",
					},
				},
			},
			Plugins: &ChartPlugins{
				Legend: &ChartLegendOptions{
					Display: false,
				},
				Tooltip: &ChartTooltipOptions{
					Enabled: true,
				},
			},
			MaintainAspectRatio: false,
			Responsive:          true,
		},
	}

	chartJSON, err := json.Marshal(chartData)
	if err != nil {
		return "", err
	}

	return string(chartJSON), nil
}

// PrepareCommandsExecutionTimeChartData prepares and returns the chart data for the command's execution time distribution.
func PrepareCommandsExecutionTimeChartData(commands []collector.Command) (string, error) {

	if commands != nil && len(commands) == 0 {
		return "", nil
	}

	var labels []string
	var data []int64
	var ids []int

	for _, cmd := range commands {
		labels = append(labels, cmd.Command)
		ids = append(ids, int(cmd.Id))
		data = append(data, cmd.ExecutionTime)
	}

	dataset := ChartDataDataset{
		Label:       "Execution Time (ms)",
		Data:        data,
		BorderWidth: 1,
	}

	chartData := ChartData{
		Type: "pie",
		Data: ChartDataData{
			Ids:      ids,
			Labels:   labels,
			Datasets: []ChartDataDataset{dataset},
		},
		Options: ChartOptions{
			Responsive:          true,
			AspectRatio:         3,
			MaintainAspectRatio: false,
			Plugins: &ChartPlugins{
				Legend: &ChartLegendOptions{
					Display:  true,
					Position: "right",
					Align:    "center",
				},
			},
		},
	}

	chartJSON, err := json.Marshal(chartData)
	if err != nil {
		return "", err
	}

	return string(chartJSON), nil
}

// PrepareCommandCategoriesExecutionTimeChartData prepares and returns the chart data for the command's execution time distribution.
func PrepareCommandCategoriesExecutionTimeChartData(commands []*collector.Command) (string, error) {

	if commands != nil && len(commands) == 0 {
		return "", nil
	}

	var labels []string
	var data []int64
	var ids []int

	for _, cmd := range commands {
		labels = append(labels, cmd.Category)
		ids = append(ids, int(cmd.Id))
		data = append(data, cmd.ExecutionTime)
	}

	dataset := ChartDataDataset{
		Label:       "Execution Time (ms)",
		Data:        data,
		BorderWidth: 1,
	}

	chartData := ChartData{
		Type: "pie",
		Data: ChartDataData{
			Ids:      ids,
			Labels:   labels,
			Datasets: []ChartDataDataset{dataset},
		},
		Options: ChartOptions{
			Responsive:          true,
			AspectRatio:         2,
			MaintainAspectRatio: false,
			Plugins: &ChartPlugins{
				Legend: &ChartLegendOptions{
					Display:  true,
					Position: "right",
					Align:    "center",
				},
			},
		},
	}

	chartJSON, err := json.Marshal(chartData)
	if err != nil {
		return "", err
	}

	return string(chartJSON), nil
}

// PrepareProcessesResourceUsageChartData prepares and returns the chart data for processes' resource usage.
func PrepareProcessesResourceUsageChartData(processes []*process.Process) (string, error) {

	if processes != nil && len(processes) == 0 {
		return "", nil
	}

	// Generate the data points for the scatter chart from processes data
	var dataPoints []DataPoint
	for _, proc := range processes {
		dataPoint := DataPoint{
			X:           proc.CPUUsage,    // x-axis represents CPU Usage
			Y:           proc.MemoryUsage, // y-axis represents Memory Usage
			ProcessName: proc.Name,
			R:           math.Sqrt(proc.CPUUsage * float64(proc.MemoryUsage)),
		}
		dataPoints = append(dataPoints, dataPoint)
	}

	datasets := []ChartDataDataset{
		{
			Label:       "Process Resource Usage",
			Data:        dataPoints,
			PointStyle:  "circle",
			PointRadius: 5, // This has to be set based on the data in JS
		},
	}

	chartData := ChartData{
		Type: "scatter",
		Data: ChartDataData{
			Datasets: datasets,
		},
		Options: ChartOptions{
			Scales: &ChartScales{
				XAxes: ChartAxisOptions{
					Title: &ChartAxisTitle{
						Display: true,
						Text:    "CPU Usage (%)",
					},
				},
				YAxes: ChartAxisOptions{
					Title: &ChartAxisTitle{
						Display: true,
						Text:    "Memory Usage (GB)",
					},
				},
			},
			Plugins: &ChartPlugins{
				Legend: &ChartLegendOptions{
					Display: true,
				},
				Tooltip: &ChartTooltipOptions{
					Enabled: true,
				},
			},
			Responsive: true,
		},
	}

	chartJSON, err := json.Marshal(chartData)
	if err != nil {
		return "", err
	}

	return string(chartJSON), nil
}
