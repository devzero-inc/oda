package database

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/devzero-inc/oda/config"
	"github.com/devzero-inc/oda/util"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// DB is the database connection.
var DB *sqlx.DB

// Setup initializes the database connection.
func Setup(odaDir string, user *user.User) {

	dbPath := filepath.Join(odaDir, "oda.db")

	db, err := sqlx.Connect("sqlite", dbPath)
	if err != nil {
		fmt.Printf("Failed to setup database: %s\n", err)
		os.Exit(1)
	}

	if err := util.ChangeFileOwnership(dbPath, user); err != nil {
		fmt.Fprintf(config.SysConfig.ErrOut, "Failed to change ownership of database: %s\n", err)
		os.Exit(1)
	}

	DB = db
}
