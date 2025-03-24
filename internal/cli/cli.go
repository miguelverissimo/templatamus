package cli

import (
	"fmt"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"templatamus/internal/model"
)

// Choose presents a list of options and returns the selected option
func Choose(prompt string, options []string) (string, error) {
	var result string
	q := &survey.Select{
		Message: prompt,
		Options: options,
	}
	return result, survey.AskOne(q, &result)
}

// MultiChoose presents a list of options and returns multiple selected options
func MultiChoose(prompt string, options []string) ([]string, error) {
	var result []string
	q := &survey.MultiSelect{
		Message: prompt,
		Options: options,
	}
	return result, survey.AskOne(q, &result)
}

// Input gets a text input from the user
func Input(prompt string) (string, error) {
	var result string
	q := &survey.Input{Message: prompt}
	return result, survey.AskOne(q, &result)
}

// InputWithDefault gets a text input from the user with a default value
func InputWithDefault(prompt string, defaultValue string) (string, error) {
	var result string
	q := &survey.Input{
		Message: prompt,
		Default: defaultValue,
	}
	return result, survey.AskOne(q, &result)
}

// Confirm asks for confirmation
func Confirm(prompt string, defaultYes bool) (bool, error) {
	var result bool
	q := &survey.Confirm{Message: prompt, Default: defaultYes}
	return result, survey.AskOne(q, &result)
}

// ExpandPath expands the ~ character to the user's home directory
func ExpandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		u, err := user.Current()
		if err != nil {
			return "", err
		}
		return filepath.Join(u.HomeDir, path[1:]), nil
	}
	return path, nil
}

// DisplayCommits shows a list of commits with their status
func DisplayCommits(commits []model.CommitInfo) {
	fmt.Println("\nAvailable commits:")
	fmt.Println("--------------------------------------------------")
	for i, commit := range commits {
		appliedStatus := ""
		if commit.IsApplied {
			appliedStatus = "[APPLIED]"
		}
		fmt.Printf("%d. %s %s\n   %s by %s on %s\n\n", 
			i+1, 
			commit.SHA[:8], 
			appliedStatus, 
			strings.Split(commit.Message, "\n")[0], 
			commit.Author, 
			commit.Date.Format("2006-01-02"))
	}
	fmt.Println("--------------------------------------------------")
}

// ChooseCommits lets the user select which commits to apply
func ChooseCommits(commits []model.CommitInfo) ([]model.CommitInfo, error) {
	DisplayCommits(commits)
	
	options := make([]string, len(commits))
	for i, commit := range commits {
		status := ""
		if commit.IsApplied {
			status = "[APPLIED] "
		}
		options[i] = fmt.Sprintf("%s %s%s", 
			commit.SHA[:8], 
			status,
			strings.Split(commit.Message, "\n")[0])
	}

	selected, err := MultiChoose("Select commits to apply (Space to select, Enter to confirm):", options)
	if err != nil {
		return nil, err
	}

	// Map selected options back to commits
	result := []model.CommitInfo{}
	for _, sel := range selected {
		for i, opt := range options {
			if sel == opt {
				result = append(result, commits[i])
				break
			}
		}
	}

	return result, nil
}

// GetDestinationPath asks the user for a destination path
func GetDestinationPath(prompt string) (string, error) {
	pathInput, err := Input(prompt)
	if err != nil {
		return "", err
	}

	// Expand ~ to home directory if needed
	if strings.HasPrefix(pathInput, "~") {
		u, err := user.Current()
		if err != nil {
			return "", err
		}
		pathInput = filepath.Join(u.HomeDir, pathInput[1:])
	}

	// Clean and resolve absolute path
	target, err := filepath.Abs(filepath.Clean(pathInput))
	if err != nil {
		return "", fmt.Errorf("invalid path: %v", err)
	}

	return target, nil
}

// DisplayConflict shows information about a conflict
func DisplayConflict(commit model.CommitInfo) {
	fmt.Println("\n⚠️  MERGE CONFLICT DETECTED ⚠️")
	fmt.Println("--------------------------------------------------")
	fmt.Printf("Commit: %s\n", commit.SHA)
	fmt.Printf("Author: %s\n", commit.Author)
	fmt.Printf("Date: %s\n", commit.Date.Format("2006-01-02 15:04:05"))
	fmt.Printf("Message: %s\n", commit.Message)
	fmt.Println("--------------------------------------------------")
	fmt.Println("Please resolve the conflicts manually and then continue.")
} 