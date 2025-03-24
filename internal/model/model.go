package model

import (
	"time"
)

// UserConfig represents the user's global configuration stored in ~/.templatamus
type UserConfig struct {
	Token string   `json:"token"`
	Repos []string `json:"repos"`
}

// ProjectMetadata represents the metadata stored in the .templatamus/metadata.json file
type ProjectMetadata struct {
	SourceRepo     string    `json:"source_repo"`
	SourceBranch   string    `json:"source_branch"`
	SourceCommit   string    `json:"source_commit"`
	CreatedAt      time.Time `json:"created_at"`
	LastSyncedAt   time.Time `json:"last_synced_at"`
	AppliedCommits []string  `json:"applied_commits"`
}

// CommitInfo represents information about a commit in the source repository
type CommitInfo struct {
	SHA       string    `json:"sha"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Date      time.Time `json:"date"`
	URL       string    `json:"url"`
	IsApplied bool      `json:"-"` // Not stored, calculated at runtime
}

// SyncStatus represents the current status of a sync operation
type SyncStatus struct {
	InProgress     bool       `json:"in_progress"`
	CurrentCommit  string     `json:"current_commit"`
	HasConflicts   bool       `json:"has_conflicts"`
	ConflictsAt    time.Time  `json:"conflicts_at"`
	ConflictCommit *CommitInfo `json:"conflict_commit,omitempty"`
} 