package util

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"

	"github.com/devzero-inc/oda/logging"
)

// FileExists checks if a file exists or not
func FileExists(filePath string) bool {
	if _, err := Fs.Stat(filePath); err == nil {
		return true
	} else if !os.IsNotExist(err) {
		logging.Log.Err(err).Msg("Failed to check if file exists or not")
	}
	return false
}

// CreateDirAndChown creates a directory and changes its ownership
func CreateDirAndChown(dirPath string, perm os.FileMode, user *user.User) error {
	if err := os.MkdirAll(dirPath, perm); err != nil {
		return err
	}

	if user != nil {
		uid, err := strconv.Atoi(user.Uid)
		if err != nil {
			return err
		}

		gid, err := strconv.Atoi(user.Gid)
		if err != nil {
			return err
		}

		if err := os.Chown(dirPath, uid, gid); err != nil {
			return err
		}
	}

	return nil
}

// WriteFileAndChown writes content to a file and changes its ownership
func WriteFileAndChown(filePath string, content []byte, perm os.FileMode, user *user.User) error {
	if err := os.WriteFile(filePath, content, perm); err != nil {
		return err
	}

	if user != nil {

		uid, err := strconv.Atoi(user.Uid)
		if err != nil {
			return err
		}

		gid, err := strconv.Atoi(user.Gid)
		if err != nil {
			return err
		}

		if err := os.Chown(filePath, uid, gid); err != nil {
			return err
		}
	}

	return nil
}

// ChangeFileOwnership changes the ownership of a file
func ChangeFileOwnership(filePath string, user *user.User) error {
	if user == nil {
		return nil
	}

	uid, err := strconv.Atoi(user.Uid)
	if err != nil {
		return err
	}

	gid, err := strconv.Atoi(user.Gid)
	if err != nil {
		return err
	}

	if err := os.Chown(filePath, uid, gid); err != nil {
		return err
	}
	return nil
}

// IsScriptPresent checks if a script is already present in a file
func IsScriptPresent(filePath, script string) bool {
	file, err := Fs.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), script) {
			return true
		}
	}
	return false
}

// AppendToFile appends content to a file
func AppendToFile(filePath, content string) error {
	f, err := Fs.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return err
	}
	return nil
}

// GetRepoNameFromConfig checks if the current directory (or any of its parent directories) is inside a Git repository.
// If inside a Git repo, it retrieves and returns the repository name.
func GetRepoNameFromConfig(path string) (string, error) {
	// Check if we're inside a Git worktree
	cmd := exec.Command("git", "-C", path, "rev-parse", "--is-inside-work-tree")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil || strings.TrimSpace(out.String()) != "true" {
		return "", fmt.Errorf("not inside a Git repository: %w", err)
	}

	// Retrieve the remote origin URL
	cmd = exec.Command("git", "-C", path, "config", "--get", "remote.origin.url")
	out.Reset()
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("could not get remote origin URL: %w", err)
	}

	url := strings.TrimSpace(out.String())
	if url == "" {
		return "", fmt.Errorf("no origin URL found in Git configuration")
	}

	// Remove .git suffix if present and extract repo name from the URL
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	repoName := parts[len(parts)-1]

	return repoName, nil
}
