package database

import (
	"fmt"
	"os"

	"github.com/devzero-inc/local-developer-analytics/config"
)

// RunMigrations all additional migrations should be registered here
func RunMigrations() {
	ensureMigrationTableExists()
	createProcessesTable()
	createCommandsTable()
	createConfigTable()
	addIndexOnProcesses()
	shellTypeToLocation()
}

func ensureMigrationTableExists() {
	createMigrationTableSQL := `
    CREATE TABLE IF NOT EXISTS schema_migrations (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        migration_name TEXT NOT NULL UNIQUE
    );`

	_, err := DB.Exec(createMigrationTableSQL)
	if err != nil {
		fmt.Fprintf(config.SysConfig.ErrOut, "Failed to create schema_migrations table: %s\n", err)
		os.Exit(1)
	}
}

func createProcessesTable() {
	migrationName := "create_processes_table"
	if !migrationApplied(migrationName) {
		createProcessesTableSQL := `
		CREATE TABLE IF NOT EXISTS processes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pid INTEGER NOT NULL,
			ppid INTEGER,
			name TEXT NOT NULL,
			status TEXT,
			created_time INTEGER,
			stored_time INTEGER,
			os TEXT,
			platform TEXT,
			platform_family TEXT,
			cpu_usage REAL,
			memory_usage REAL
		);`

		_, err := DB.Exec(createProcessesTableSQL)
		if err != nil {
			fmt.Fprintf(config.SysConfig.ErrOut, "Failed to create processes table: %s\n", err)
			os.Exit(1)
		}
		recordMigration(migrationName)
	}
}

func createCommandsTable() {
	migrationName := "create_commands_table"
	if !migrationApplied(migrationName) {
		createCommandsTableSQL := `
		CREATE TABLE IF NOT EXISTS commands (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			category TEXT NOT NULL,
			command TEXT NOT NULL,
			user TEXT,
			directory TEXT,
			execution_time INTEGER,
			start_time INTEGER,
			end_time INTEGER,
			status TEXT,
			result TEXT,
			repository TEXT
		);`

		_, err := DB.Exec(createCommandsTableSQL)
		if err != nil {
			fmt.Fprintf(config.SysConfig.ErrOut, "Failed to create commands table: %s\n", err)
			os.Exit(1)
		}
		recordMigration(migrationName)
	}
}

func addIndexOnProcesses() {
	migrationName := "add_index_on_processes"
	if !migrationApplied(migrationName) {
		indexesSQL := []string{
			`CREATE INDEX IF NOT EXISTS idx_processes_time_cpu_memory ON processes(stored_time, cpu_usage, memory_usage);`,
			`CREATE INDEX IF NOT EXISTS idx_processes_pid_name ON processes(pid, name);`,
		}

		for _, sql := range indexesSQL {
			_, err := DB.Exec(sql)
			if err != nil {
				fmt.Fprintf(config.SysConfig.ErrOut, "Failed to create index: %s\n", err)
				os.Exit(1)
			}
		}
		recordMigration(migrationName)
	}
}

func createConfigTable() {
	migrationName := "create_config_table"
	if !migrationApplied(migrationName) {
		createConfigTableSQL := `
		CREATE TABLE IF NOT EXISTS config (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			os TEXT NOT NULL,
			os_name TEXT NOT NULL,
			home_dir TEXT NOT NULL,
			oda_dir TEXT NOT NULL,
			is_root BOOLEAN NOT NULL,
			exe_path TEXT NOT NULL
		);`

		_, err := DB.Exec(createConfigTableSQL)
		if err != nil {
			fmt.Fprintf(config.SysConfig.ErrOut, "Failed to create config table: %s\n", err)
			os.Exit(1)
		}
		recordMigration(migrationName)
	}
}

func shellTypeToLocation() {
	migrationName := "shell_type_to_location"
	if !migrationApplied(migrationName) {
		shellTypeToLocationSQL := `
		CREATE TABLE IF NOT EXISTS shell_type_to_location (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			shell_type INTEGER NOT NULL,
			shell_location TEXT NOT NULL,
			config_id INTEGER NOT NULL REFERENCES config(id)
		);`

		_, err := DB.Exec(shellTypeToLocationSQL)
		if err != nil {
			fmt.Fprintf(config.SysConfig.ErrOut, "Failed to add shell_type_to_location column: %s\n", err)
			os.Exit(1)
		}
		recordMigration(migrationName)
	}
}

func migrationApplied(migrationName string) bool {
	var count int
	err := DB.Get(&count, "SELECT COUNT(*) FROM schema_migrations WHERE migration_name = ?", migrationName)
	if err != nil {
		fmt.Fprintf(config.SysConfig.ErrOut, "Failed to query schema_migrations table: %s\n", err)
		os.Exit(1)
	}
	return count > 0
}

func recordMigration(migrationName string) {
	_, err := DB.Exec("INSERT INTO schema_migrations (migration_name) VALUES (?)", migrationName)
	if err != nil {
		fmt.Fprintf(config.SysConfig.ErrOut, "Failed to record migration: %s\n", err)
		os.Exit(1)
	}
}
