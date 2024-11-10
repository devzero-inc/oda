package collector

import (
	"regexp"
	"time"

	"github.com/devzero-inc/oda/config"
	"github.com/devzero-inc/oda/database"
	gen "github.com/devzero-inc/oda/gen/api/v1"
	"github.com/devzero-inc/oda/logging"
)

// Command is the model for command
type Command struct {
	Id            int64  `json:"id" db:"id"`
	Category      string `json:"category" db:"category"`
	Command       string `json:"command" db:"command"`
	User          string `json:"user" db:"user"`
	Directory     string `json:"directory" db:"directory"`
	ExecutionTime int64  `json:"execution_time" db:"execution_time"`
	StartTime     int64  `json:"start_time" db:"start_time"`
	EndTime       int64  `json:"end_time" db:"end_time"`
	Status        string `json:"status" db:"status"`
	Result        string `json:"result" db:"result"`
	Repository    string `json:"repository" db:"repository"`
	PID           int64  `json:"pid" db:"pid"`
}

// GetCommandById fetches a command by its ID
func GetCommandById(id int64) (*Command, error) {
	var command Command
	query := `SELECT * FROM commands WHERE id = ?`

	if err := database.DB.Get(&command, query, id); err != nil {
		logging.Log.Err(err).Msg("Failed to get command by id")
		return nil, err
	}

	return &command, nil
}

// GetAllCommandsForPeriod fetches all commands for a given period
func GetAllCommandsForPeriod(start int64, end int64) ([]*Command, error) {
	var commands []*Command

	query := `SELECT id, category, SUM(execution_time) AS execution_time 
              FROM commands 
              WHERE start_time BETWEEN ? AND ? 
              GROUP BY category 
              ORDER BY category ASC, SUM(execution_time) DESC;`

	if err := database.DB.Select(&commands, query, start, end); err != nil {
		logging.Log.Err(err).Msg("Failed to get aggregated commands with start and end times")
		return nil, err
	}

	return commands, nil
}

// GetAllCommandsForCategoryForPeriod fetches all commands for a given category and period
func GetAllCommandsForCategoryForPeriod(category string, start int64, end int64) ([]Command, error) {
	var commands []Command

	query := `SELECT id, category, command, SUM(execution_time) AS execution_time 
              FROM commands 
              WHERE category = ? AND start_time BETWEEN ? AND ? 
              GROUP BY command 
              ORDER BY command ASC, SUM(execution_time) DESC;`

	if err := database.DB.Select(&commands, query, category, start, end); err != nil {
		logging.Log.Err(err).Msg("Failed to get aggregated commands with start and end times")
		return nil, err
	}

	return commands, nil
}

// DeleteCommandsByDays deletes records older than n days
func DeleteCommandsByDays(days int) error {
	// Calculate the time when old records will be deleted
	timeToDelete := time.Now().AddDate(0, 0, -days).Unix()

	result, err := database.DB.Exec("DELETE FROM commands WHERE stored_time < ?", timeToDelete)
	if err != nil {
		return err
	}

	_, err = result.RowsAffected()

	return err
}

// InsertCommand inserts a command into the database
func InsertCommand(command Command) error {
	query := `INSERT INTO commands (category, command, user, directory, execution_time, start_time, end_time, status, result, repository, pid)
	VALUES (:category, :command, :user, :directory, :execution_time, :start_time, :end_time, :status, :result, :repository, :pid)`

	_, err := database.DB.NamedExec(query, command)

	return err
}

// ParseCommand extracts the command name from a command string.
func ParseCommand(command string) string {

	// TODO: there might be some other cases as well like: watch, time etc
	var pattern = regexp.MustCompile(`^(?:sudo|nohup)?\s*(?:\./|/usr/bin/|/bin/|/usr/local/bin/)?([^/ ]+?)(?:\s|$)`)

	matches := pattern.FindStringSubmatch(command)
	if len(matches) > 1 {
		return matches[1]
	}

	return command
}

// IsCommandAcceptable checks if a command string matches a configured regex pattern.
// Commands that match the regex are considered unacceptable, and it returns false.
// If the regex is empty or the command does not match, it returns true.
func IsCommandAcceptable(command string, excludeRegex string) bool {
	if excludeRegex != "" {
		logging.Log.Debug().Msgf("Checking if command %s is acceptable for regex: %s", command, config.AppConfig.ExcludeRegex)
		var pattern = regexp.MustCompile(excludeRegex)
		return !pattern.MatchString(command)
	}

	return true
}

func MapCommandToProto(command Command) *gen.Command {
	return &gen.Command{
		Id:            command.Id,
		Category:      command.Category,
		Command:       command.Command,
		User:          command.User,
		Directory:     command.Directory,
		ExecutionTime: command.ExecutionTime,
		StartTime:     command.StartTime,
		EndTime:       command.EndTime,
		Status:        command.Status,
		Result:        command.Result,
		Repository:    command.Repository,
		Pid:           command.PID,
	}
}
