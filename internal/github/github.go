package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"templatamus/internal/model"
)

// Client represents a GitHub API client
type Client struct {
	Token string
}

// NewClient creates a new GitHub client
func NewClient(token string) *Client {
	return &Client{Token: token}
}

// GetTags retrieves all tags for a repository
func (c *Client) GetTags(owner, repo string) ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/tags", owner, repo)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s: %s", resp.Status, body)
	}

	var tags []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}

	result := []string{}
	for _, tag := range tags {
		result = append(result, tag.Name)
	}
	return result, nil
}

// GetBranches retrieves all branches for a repository
func (c *Client) GetBranches(owner, repo string) ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/branches", owner, repo)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s: %s", resp.Status, body)
	}

	var branches []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&branches); err != nil {
		return nil, err
	}

	result := []string{}
	for _, b := range branches {
		result = append(result, b.Name)
	}
	return result, nil
}

// GetDefaultBranch retrieves the default branch for a repository
func (c *Client) GetDefaultBranch(owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API error: %s: %s", resp.Status, body)
	}

	var data struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	return data.DefaultBranch, nil
}

// DownloadZip downloads a repository as a zip archive
func (c *Client) DownloadZip(owner, repo, ref string) ([]byte, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/zipball/%s", owner, repo, ref)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s: %s", resp.Status, body)
	}

	return io.ReadAll(resp.Body)
}

// GetCommits retrieves commits for a repository
func (c *Client) GetCommits(owner, repo, branch string, since time.Time) ([]model.CommitInfo, error) {
	// We'll use per_page=100 to get more commits in one response
	// NOTE: This is limited to the first 100 commits, which should be enough for most cases
	// For repositories with more commits, we'd need to implement pagination
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits?sha=%s&per_page=100", owner, repo, branch)
	
	// Add since parameter if provided and not zero
	if !since.IsZero() {
		url += fmt.Sprintf("&since=%s", since.Format(time.RFC3339))
	}
	
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s: %s", resp.Status, body)
	}

	var ghCommits []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name  string    `json:"name"`
				Date  time.Time `json:"date"`
			} `json:"author"`
		} `json:"commit"`
		HTMLURL string `json:"html_url"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&ghCommits); err != nil {
		return nil, err
	}

	commits := make([]model.CommitInfo, 0, len(ghCommits))
	for _, c := range ghCommits {
		commits = append(commits, model.CommitInfo{
			SHA:     c.SHA,
			Message: c.Commit.Message,
			Author:  c.Commit.Author.Name,
			Date:    c.Commit.Author.Date,
			URL:     c.HTMLURL,
		})
	}

	return commits, nil
}

// GetCommit retrieves a single commit
func (c *Client) GetCommit(owner, repo, sha string) (*model.CommitInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", owner, repo, sha)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s: %s", resp.Status, body)
	}

	var ghCommit struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name  string    `json:"name"`
				Date  time.Time `json:"date"`
			} `json:"author"`
		} `json:"commit"`
		HTMLURL string `json:"html_url"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&ghCommit); err != nil {
		return nil, err
	}

	return &model.CommitInfo{
		SHA:     ghCommit.SHA,
		Message: ghCommit.Commit.Message,
		Author:  ghCommit.Commit.Author.Name,
		Date:    ghCommit.Commit.Author.Date,
		URL:     ghCommit.HTMLURL,
	}, nil
}

// GetDiff gets the diff for a commit
func (c *Client) GetDiff(owner, repo, sha string) ([]byte, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", owner, repo, sha)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+c.Token)
	req.Header.Set("Accept", "application/vnd.github.diff")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s: %s", resp.Status, body)
	}

	return io.ReadAll(resp.Body)
}

// GetJSON performs a GET request to the GitHub API and unmarshals the response JSON into the provided object
func (c *Client) GetJSON(url string, v interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	
	req.Header.Set("Authorization", "token "+c.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API error: %s: %s", resp.Status, body)
	}

	return json.NewDecoder(resp.Body).Decode(v)
} 