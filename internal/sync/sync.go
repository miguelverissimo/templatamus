package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"templatamus/internal/cli"
	"templatamus/internal/config"
	"templatamus/internal/git"
	"templatamus/internal/github"
	"templatamus/internal/model"
)

// DetectProject checks if the current directory or specified directory is a templatamus project
func DetectProject() (string, bool, error) {
	// First check current directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", false, fmt.Errorf("failed to get current directory: %w", err)
	}

	if config.HasProjectMetadata(cwd) {
		return cwd, true, nil
	}

	// Ask user for path
	fmt.Println("No templatamus project found in current directory.")
	target, err := cli.GetDestinationPath("Where is your project located? (or provide a new path for a new project)")
	if err != nil {
		return "", false, err
	}

	// Check if the target path exists and has metadata
	if _, err := os.Stat(target); err == nil {
		if config.HasProjectMetadata(target) {
			return target, true, nil
		}
		// Target exists but is not a templatamus project
		isNew, err := cli.Confirm(fmt.Sprintf("Directory %s exists but is not a templatamus project. Create a new project there?", target), false)
		if err != nil {
			return "", false, err
		}
		if isNew {
			return target, false, nil
		}
		return "", false, fmt.Errorf("aborted by user")
	}

	// Target doesn't exist, must be new project
	return target, false, nil
}

// SyncProject synchronizes a project with its source repository
func SyncProject(dir string, ghClient *github.Client) error {
	// Load metadata
	metadata, err := config.LoadProjectMetadata(dir)
	if err != nil {
		return fmt.Errorf("failed to load project metadata: %w", err)
	}

	// Debug: Show current metadata
	fmt.Printf("Project metadata: source=%s, branch=%s\n", metadata.SourceRepo, metadata.SourceBranch)
	fmt.Printf("Original source commit: %s\n", metadata.SourceCommit[:8])
	fmt.Printf("Applied commits: %d\n", len(metadata.AppliedCommits))

	// Check if there's a sync in progress
	syncStatus, err := config.LoadSyncStatus(dir)
	if err != nil {
		return fmt.Errorf("failed to load sync status: %w", err)
	}

	// If there's a sync in progress with conflicts, handle it
	if syncStatus.InProgress && syncStatus.HasConflicts {
		return handleConflictResolution(dir, metadata, syncStatus)
	}

	// Split repo into owner/name
	parts := strings.Split(metadata.SourceRepo, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository format: %s", metadata.SourceRepo)
	}
	owner, repo := parts[0], parts[1]

	// Get all commits from the branch
	fmt.Println("Checking for updates...")
	commits, err := ghClient.GetCommits(owner, repo, metadata.SourceBranch, time.Time{}) // Get all commits
	if err != nil {
		return fmt.Errorf("failed to get commits: %w", err)
	}

	fmt.Printf("Found %d commits from GitHub\n", len(commits))

	// Create a map of applied commits for quick lookup
	appliedSet := make(map[string]bool)
	for _, sha := range metadata.AppliedCommits {
		appliedSet[sha] = true
	}

	// Filter for only new commits that haven't been applied
	sourceCommitFound := false
	var newCommits []model.CommitInfo
	
	// First, sort commits by date (oldest first)
	sort.Slice(commits, func(i, j int) bool {
		return commits[i].Date.Before(commits[j].Date)
	})
	
	// Find the index of the source commit
	sourceCommitIndex := -1
	for i, commit := range commits {
		if commit.SHA == metadata.SourceCommit {
			sourceCommitIndex = i
			sourceCommitFound = true
			break
		}
	}
	
	if !sourceCommitFound {
		fmt.Printf("Warning: Source commit %s not found in commit history.\n", metadata.SourceCommit[:8])
		// If we can't find the source commit, we'll include all commits that haven't been applied
		for _, commit := range commits {
			if !appliedSet[commit.SHA] {
				newCommits = append(newCommits, commit)
			}
		}
	} else {
		// Include only commits that come after the source commit
		for i := sourceCommitIndex + 1; i < len(commits); i++ {
			commit := commits[i]
			if !appliedSet[commit.SHA] {
				newCommits = append(newCommits, commit)
			}
		}
	}

	if len(newCommits) == 0 {
		fmt.Println("Project is already up to date (no new commits found).")
		return nil
	}

	fmt.Printf("Found %d new commits that haven't been applied.\n", len(newCommits))

	// Let user select commits to apply
	selectedCommits, err := cli.ChooseCommits(newCommits)
	if err != nil {
		return fmt.Errorf("commit selection failed: %w", err)
	}

	if len(selectedCommits) == 0 {
		fmt.Println("No commits selected. Aborting sync.")
		return nil
	}

	// Apply each selected commit
	for _, commit := range selectedCommits {
		// Double-check it's not already applied
		if appliedSet[commit.SHA] {
			fmt.Printf("Skipping already applied commit: %s\n", commit.SHA[:8])
			continue
		}

		fmt.Printf("Applying commit: %s - %s\n", commit.SHA[:8], strings.Split(commit.Message, "\n")[0])

		// Get the diff
		diff, err := ghClient.GetDiff(owner, repo, commit.SHA)
		if err != nil {
			return fmt.Errorf("failed to get diff for commit %s: %w", commit.SHA, err)
		}

		// Apply the diff
		success, err := git.ApplyDiff(dir, diff)
		if err != nil {
			return fmt.Errorf("failed to apply diff: %w", err)
		}

		if !success {
			// Save the patch file
			patchPath := filepath.Join(dir, ".templatamus", "conflict.patch")
			if err := os.MkdirAll(filepath.Dir(patchPath), 0755); err != nil {
				return fmt.Errorf("failed to create .templatamus directory: %w", err)
			}
			if err := os.WriteFile(patchPath, []byte(diff), 0644); err != nil {
				return fmt.Errorf("failed to save patch file: %w", err)
			}

			// Save the conflict status
			syncStatus.InProgress = true
			syncStatus.CurrentCommit = commit.SHA
			syncStatus.HasConflicts = true
			syncStatus.ConflictsAt = time.Now()
			syncStatus.ConflictCommit = &commit

			if err := config.SaveSyncStatus(dir, syncStatus); err != nil {
				return fmt.Errorf("failed to save sync status: %w", err)
			}

			// Display conflict information and instructions
			fmt.Printf("\nMerge conflicts detected while applying commit %s\n", commit.SHA[:8])
			fmt.Printf("Commit message: %s\n", strings.Split(commit.Message, "\n")[0])
			fmt.Printf("Author: %s\n", commit.Author)
			fmt.Printf("Date: %s\n\n", commit.Date.Format(time.RFC3339))
			
			fmt.Println("To resolve the conflicts:")
			fmt.Println("1. The patch file has been saved to .templatamus/conflict.patch")
			fmt.Println("2. Review the conflicts in your working directory")
			fmt.Println("3. Resolve the conflicts manually")
			fmt.Println("4. Stage and commit your changes")
			fmt.Println("5. Run 'templatamus' again to continue the sync")
			fmt.Println("\nOr if you want to skip this commit:")
			fmt.Println("1. Run 'git reset --hard HEAD' to discard changes")
			fmt.Println("2. Run 'templatamus' again to continue with the next commit")
			
			return fmt.Errorf("merge conflicts detected, please resolve manually and run templatamus again")
		}

		// Commit the changes
		commitMsg := fmt.Sprintf("Synced with %s: %s", metadata.SourceRepo, strings.Split(commit.Message, "\n")[0])
		if err := git.CommitChanges(dir, commitMsg); err != nil {
			return fmt.Errorf("failed to commit changes: %w", err)
		}

		// Update metadata
		metadata.AppliedCommits = append(metadata.AppliedCommits, commit.SHA)
		metadata.LastSyncedAt = time.Now()

		if err := config.SaveProjectMetadata(dir, metadata); err != nil {
			return fmt.Errorf("failed to update metadata: %w", err)
		}

		fmt.Printf("Successfully applied commit: %s\n", commit.SHA[:8])
	}

	fmt.Println("Sync completed successfully.")
	return nil
}

// handleConflictResolution handles resolving conflicts from a previous sync
func handleConflictResolution(dir string, metadata *model.ProjectMetadata, syncStatus *model.SyncStatus) error {
	if syncStatus.ConflictCommit == nil {
		return fmt.Errorf("missing conflict commit information")
	}

	commit := *syncStatus.ConflictCommit
	fmt.Printf("Detected a previous sync with conflicts for commit %s\n", commit.SHA[:8])
	
	// Check if they want to consider the conflict resolved
	resolved, err := cli.Confirm("Have you resolved the conflicts and want to continue?", true)
	if err != nil {
		return err
	}

	if !resolved {
		// Ask if they want to abort this commit and move on
		abort, err := cli.Confirm("Do you want to abort applying this commit and mark it as skipped?", false)
		if err != nil {
			return err
		}

		if abort {
			// Clear the sync status
			if err := config.ClearSyncStatus(dir); err != nil {
				return fmt.Errorf("failed to clear sync status: %w", err)
			}
			fmt.Printf("Skipped commit %s due to unresolved conflicts.\n", commit.SHA[:8])
			return nil
		}

		return fmt.Errorf("sync aborted, please resolve conflicts and try again")
	}

	// Commit the resolved changes
	commitMsg := fmt.Sprintf("Synced with %s: %s (resolved conflicts)", metadata.SourceRepo, strings.Split(commit.Message, "\n")[0])
	if err := git.CommitChanges(dir, commitMsg); err != nil {
		return fmt.Errorf("failed to commit resolved changes: %w", err)
	}

	// Update metadata
	metadata.AppliedCommits = append(metadata.AppliedCommits, commit.SHA)
	metadata.LastSyncedAt = time.Now()

	if err := config.SaveProjectMetadata(dir, metadata); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	// Clear the sync status
	if err := config.ClearSyncStatus(dir); err != nil {
		return fmt.Errorf("failed to clear sync status: %w", err)
	}

	fmt.Printf("Successfully applied commit %s with resolved conflicts.\n", commit.SHA[:8])
	return nil
}

// CreateProjectFromZip creates a new project from a downloaded zip
func CreateProjectFromZip(zipData []byte, targetDir, repoFull, branch, commit string) error {
	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "templatamus-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract to temporary directory
	if err := git.ExtractZip(zipData, tempDir); err != nil {
		return fmt.Errorf("extract failed: %w", err)
	}

	// Find root directory in the extracted content
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return fmt.Errorf("failed to read temp directory: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("empty zip file")
	}

	rootDir := ""
	for _, entry := range entries {
		if entry.IsDir() {
			rootDir = filepath.Join(tempDir, entry.Name())
			break
		}
	}

	if rootDir == "" {
		return fmt.Errorf("no root directory found in zip")
	}

	// Create target directory
	if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Move content to target directory
	if err := git.MoveDirContents(rootDir, targetDir); err != nil {
		return fmt.Errorf("failed to move content: %w", err)
	}

	// Create metadata
	if err := config.CreateInitialMetadata(targetDir, repoFull, branch, commit); err != nil {
		return fmt.Errorf("failed to create metadata: %w", err)
	}

	return nil
} 