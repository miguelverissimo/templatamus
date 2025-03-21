package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

type Config struct {
	Token string   `json:"token"`
	Repos []string `json:"repos"`
}

func loadConfig() (*Config, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(u.HomeDir, ".templatamus")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func choose(prompt string, options []string) (string, error) {
	var result string
	q := &survey.Select{
		Message: prompt,
		Options: options,
	}
	return result, survey.AskOne(q, &result)
}

func input(prompt string) (string, error) {
	var result string
	q := &survey.Input{Message: prompt}
	return result, survey.AskOne(q, &result)
}

func confirm(prompt string, defaultYes bool) (bool, error) {
	var result bool
	q := &survey.Confirm{Message: prompt, Default: defaultYes}
	return result, survey.AskOne(q, &result)
}

func getTags(token, owner, repo string) ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/tags", owner, repo)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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

func getBranches(token, owner, repo string) ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/branches", owner, repo)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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

func getDefaultBranch(token, owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	return data.DefaultBranch, nil
}

func downloadZip(token, owner, repo, ref string) ([]byte, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/zipball/%s", owner, repo, ref)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func unzipToDir(data []byte, dir string) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, f := range r.File {
		fpath := filepath.Join(dir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(fpath), 0755)
		in, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(fpath)
		if err != nil {
			in.Close()
			return err
		}
		io.Copy(out, in)
		in.Close()
		out.Close()
	}
	return nil
}

func gitInitCommit(dir, msg string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = exec.Command("git", "commit", "-m", msg)
	cmd.Dir = dir
	return cmd.Run()
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	fmt.Println("Templatamus 1.0")
	repoFull, err := choose("Choose the repo", cfg.Repos)
	if err != nil {
		log.Fatal(err)
	}
	parts := strings.Split(repoFull, "/")
	owner, repo := parts[0], parts[1]

	fmt.Printf("You're creating an app from the %s repository\n", repoFull)

	var ref string
	choice, _ := choose("Do you want to pull head, branch or tag?", []string{"head", "branch", "tag"})
	switch choice {
	case "head":
		ref, err = getDefaultBranch(cfg.Token, owner, repo)
		if err != nil {
			log.Fatal("Failed to get default branch:", err)
		}
	case "branch":
		branches, err := getBranches(cfg.Token, owner, repo)
		if err != nil {
			log.Fatal("Failed to get branches:", err)
		}
		ref, err = choose("Choose a branch", branches)
		if err != nil {
			log.Fatal(err)
		}
	case "tag":
		tags, err := getTags(cfg.Token, owner, repo)
		if err != nil || len(tags) == 0 {
			log.Fatalf("Failed to fetch tags: %v", err)
		}
		ref, err = choose("Choose a tag to download", tags)
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Printf("You're creating an app from %s@%s\n", repoFull, ref)
	pathInput, err := input("Where do you want to create the project? (e.g., myrepo, ../foo, ~/projects/bar)")
	if err != nil {
		log.Fatal(err)
	}

	// Expand ~ to home dir if needed
	if strings.HasPrefix(pathInput, "~") {
		u, _ := user.Current()
		pathInput = filepath.Join(u.HomeDir, pathInput[1:])
	}

	// Clean and resolve absolute path
	target, err := filepath.Abs(filepath.Clean(pathInput))
	if err != nil {
		log.Fatalf("Invalid path: %v", err)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		log.Fatalf("Failed to create parent directory: %v", err)
	}

	fmt.Println("Downloading...")
	zipData, err := downloadZip(cfg.Token, owner, repo, ref)
	if err != nil {
		log.Fatalf("Failed to download zip: %v", err)
	}

	fmt.Println("Unzipping...")
	temp := filepath.Join(os.TempDir(), "templatamus-unzip")
	os.MkdirAll(temp, 0755)
	if err := unzipToDir(zipData, temp); err != nil {
		log.Fatalf("Unzip failed: %v", err)
	}

	dirs, _ := os.ReadDir(temp)
	if len(dirs) > 0 && dirs[0].IsDir() {
		if err := os.Rename(filepath.Join(temp, dirs[0].Name()), target); err != nil {
			log.Fatalf("Failed to move project: %v", err)
		}
	} else {
		log.Fatal("Unexpected zip structure")
	}

	fmt.Println("Done.")

	if ok, _ := confirm("Do you want to init a git repo and initial commit?", true); ok {
		if err := gitInitCommit(target, fmt.Sprintf("initial commit from %s@%s", repoFull, ref)); err != nil {
			log.Fatalf("Git init failed: %v", err)
		}
		fmt.Println("Committed.")
	}

	fmt.Println("done!")
}
