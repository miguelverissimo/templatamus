package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"templatamus/internal/model"
)

const (
	metadataDir  = ".templatamus"
	metadataFile = "metadata.json"
	syncFile     = "sync.json"
)

// LoadUserConfig loads the user's configuration from ~/.templatamus
func LoadUserConfig() (*model.UserConfig, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(u.HomeDir, ".templatamus")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg model.UserConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// HasProjectMetadata checks if the given directory has templatamus metadata
func HasProjectMetadata(dir string) bool {
	metadataPath := filepath.Join(dir, metadataDir, metadataFile)
	_, err := os.Stat(metadataPath)
	return err == nil
}

// LoadProjectMetadata loads the project metadata from the .templatamus directory
func LoadProjectMetadata(dir string) (*model.ProjectMetadata, error) {
	metadataPath := filepath.Join(dir, metadataDir, metadataFile)
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata model.ProjectMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &metadata, nil
}

// SaveProjectMetadata saves the project metadata to the .templatamus directory
func SaveProjectMetadata(dir string, metadata *model.ProjectMetadata) error {
	metadataDir := filepath.Join(dir, metadataDir)
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	metadataPath := filepath.Join(metadataDir, metadataFile)
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// CreateInitialMetadata creates the initial metadata for a new project
func CreateInitialMetadata(dir, repo, branch, commit string) error {
	metadata := &model.ProjectMetadata{
		SourceRepo:     repo,
		SourceBranch:   branch,
		SourceCommit:   commit,
		CreatedAt:      time.Now(),
		LastSyncedAt:   time.Now(),
		AppliedCommits: []string{commit},
	}

	return SaveProjectMetadata(dir, metadata)
}

// LoadSyncStatus loads the current sync status
func LoadSyncStatus(dir string) (*model.SyncStatus, error) {
	syncPath := filepath.Join(dir, metadataDir, syncFile)
	
	// If file doesn't exist, return empty status
	if _, err := os.Stat(syncPath); os.IsNotExist(err) {
		return &model.SyncStatus{}, nil
	}
	
	data, err := os.ReadFile(syncPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sync status: %w", err)
	}

	var status model.SyncStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("failed to parse sync status: %w", err)
	}

	return &status, nil
}

// SaveSyncStatus saves the current sync status
func SaveSyncStatus(dir string, status *model.SyncStatus) error {
	metadataDir := filepath.Join(dir, metadataDir)
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	syncPath := filepath.Join(metadataDir, syncFile)
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sync status: %w", err)
	}

	if err := os.WriteFile(syncPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write sync status: %w", err)
	}

	return nil
}

// ClearSyncStatus clears the sync status
func ClearSyncStatus(dir string) error {
	syncPath := filepath.Join(dir, metadataDir, syncFile)
	if _, err := os.Stat(syncPath); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(syncPath)
} 