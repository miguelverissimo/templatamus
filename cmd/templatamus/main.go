package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"templatamus/internal/cli"
	"templatamus/internal/config"
	"templatamus/internal/git"
	"templatamus/internal/github"
	"templatamus/internal/model"
	"templatamus/internal/sync"
)

// These variables are set during compilation
var (
	Version   = "dev"
	BuildDate = time.Now().Format(time.RFC3339)
)

func main() {
	Main()
}

// Main is the entry point of the application
func Main() {
	// Display version information
	fmt.Printf("Templatamus v%s (built %s)\n\n", Version, BuildDate)

	// Load user configuration
	cfg, err := config.LoadUserConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create GitHub client
	client := github.NewClient(cfg.Token)

	// Detect project
	dir, isExisting, err := sync.DetectProject()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if !isExisting {
		// Create new project
		if err := createNewProject(dir, cfg, client); err != nil {
			log.Fatalf("Error: %v", err)
		}
	} else {
		// Sync existing project
		if err := sync.SyncProject(dir, client); err != nil {
			log.Fatalf("Error: %v", err)
		}
	}
}

// getCommitSHAForTag gets the commit SHA for a tag
func getCommitSHAForTag(client *github.Client, owner, repo, tag string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/refs/tags/%s", owner, repo, tag)
	
	var tagRef struct {
		Object struct {
			SHA string `json:"sha"`
			Type string `json:"type"`
			URL string `json:"url"`
		} `json:"object"`
	}
	
	if err := client.GetJSON(url, &tagRef); err != nil {
		return "", fmt.Errorf("failed to get tag reference: %w", err)
	}
	
	// If it's a tag object, we need to get the commit it points to
	if tagRef.Object.Type == "tag" {
		var tagObj struct {
			Object struct {
				SHA string `json:"sha"`
			} `json:"object"`
		}
		
		if err := client.GetJSON(tagRef.Object.URL, &tagObj); err != nil {
			return "", fmt.Errorf("failed to get tag object: %w", err)
		}
		
		return tagObj.Object.SHA, nil
	}
	
	// It's a direct reference to a commit
	return tagRef.Object.SHA, nil
}

// createNewProject handles creating a new project
func createNewProject(targetDir string, cfg *model.UserConfig, ghClient *github.Client) error {
	// Choose repo
	repoFull, err := cli.Choose("Choose the repo", cfg.Repos)
	if err != nil {
		return err
	}
	
	parts := strings.Split(repoFull, "/")
	owner, repo := parts[0], parts[1]

	fmt.Printf("You're creating an app from the %s repository\n", repoFull)

	// Choose reference (head, branch, tag)
	var ref, commitSHA string
	choice, err := cli.Choose("Do you want to pull head, branch or tag?", []string{"head", "branch", "tag"})
	if err != nil {
		return err
	}

	switch choice {
	case "head":
		ref, err = ghClient.GetDefaultBranch(owner, repo)
		if err != nil {
			return fmt.Errorf("failed to get default branch: %w", err)
		}
		
		// Get the latest commit on the branch
		commits, err := ghClient.GetCommits(owner, repo, ref, time.Time{})
		if err != nil {
			return fmt.Errorf("failed to get commits: %w", err)
		}
		
		if len(commits) > 0 {
			commitSHA = commits[0].SHA
		} else {
			return fmt.Errorf("no commits found on branch %s", ref)
		}
		
	case "branch":
		branches, err := ghClient.GetBranches(owner, repo)
		if err != nil {
			return fmt.Errorf("failed to get branches: %w", err)
		}
		ref, err = cli.Choose("Choose a branch", branches)
		if err != nil {
			return err
		}
		
		// Get the latest commit on the branch
		commits, err := ghClient.GetCommits(owner, repo, ref, time.Time{})
		if err != nil {
			return fmt.Errorf("failed to get commits: %w", err)
		}
		
		if len(commits) > 0 {
			commitSHA = commits[0].SHA
		} else {
			return fmt.Errorf("no commits found on branch %s", ref)
		}
		
	case "tag":
		tags, err := ghClient.GetTags(owner, repo)
		if err != nil {
			return fmt.Errorf("failed to get tags: %w", err)
		}
		if len(tags) == 0 {
			return fmt.Errorf("no tags found in repository")
		}
		ref, err = cli.Choose("Choose a tag to download", tags)
		if err != nil {
			return err
		}
		
		// Get the commit SHA that this tag points to
		commitSHA, err = getCommitSHAForTag(ghClient, owner, repo, ref)
		if err != nil {
			// If we can't get the exact commit SHA, use the tag as a fallback
			fmt.Printf("Warning: Could not resolve tag to commit: %v\n", err)
			commitSHA = ref
		}
	}

	fmt.Printf("You're creating an app from %s@%s (commit: %s)\n", repoFull, ref, commitSHA[:8])

	// Download zip
	fmt.Println("Downloading...")
	zipData, err := ghClient.DownloadZip(owner, repo, ref)
	if err != nil {
		return fmt.Errorf("failed to download zip: %w", err)
	}

	// Create project from zip
	fmt.Println("Unzipping...")
	if err := sync.CreateProjectFromZip(zipData, targetDir, repoFull, ref, commitSHA); err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	// Initialize git repository if requested
	ok, err := cli.Confirm("Do you want to init a git repo and initial commit?", true)
	if err != nil {
		return err
	}

	if ok {
		commitMsg := fmt.Sprintf("Initial commit from %s@%s", repoFull, ref)
		if err := git.InitRepo(targetDir, commitMsg); err != nil {
			return fmt.Errorf("git init failed: %w", err)
		}
		fmt.Printf("Committed: %s\n", commitMsg)
	}

	return nil
} 