package git

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// InitRepo initializes a git repository in the specified directory
func InitRepo(dir, msg string) error {
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

// ApplyDiff applies a diff to the repository
// Returns true if the diff was applied successfully, false if there are conflicts
func ApplyDiff(dir string, diff []byte) (bool, error) {
	// Write diff to a temporary file
	tmpFile, err := os.CreateTemp("", "templatamus-diff-*.patch")
	if err != nil {
		return false, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(diff); err != nil {
		return false, fmt.Errorf("failed to write diff: %w", err)
	}
	tmpFile.Close()

	// Apply the patch
	cmd := exec.Command("git", "apply", "--reject", "--whitespace=fix", tmpFile.Name())
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Check if there were conflicts
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Look for .rej files to determine if there were actual conflicts
			rejFiles, err := filepath.Glob(filepath.Join(dir, "*.rej"))
			if err != nil {
				return false, fmt.Errorf("failed to check for .rej files: %w", err)
			}
			if len(rejFiles) > 0 {
				return false, nil // Conflicts detected
			}
		}
		return false, fmt.Errorf("failed to apply patch: %w", err)
	}

	return true, nil
}

// CommitChanges commits the changes with the given message
func CommitChanges(dir, msg string) error {
	// Add all changes
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// Check if there are changes to commit
	cmd = exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = dir
	err := cmd.Run()
	
	// Exit code 1 means there are changes, which is good in this case
	if err == nil {
		// No changes to commit
		return nil
	}

	// Commit the changes
	cmd = exec.Command("git", "commit", "-m", msg)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	return nil
}

// ExtractZip extracts a zip file to the specified directory
func ExtractZip(data []byte, dir string) error {
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

// MoveDirContents moves the contents of the source directory to the target directory
func MoveDirContents(src, dst string) error {
	// Read the source directory entries
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Create the destination directory if it doesn't exist
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	// Move each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if err := os.Rename(srcPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}

// CheckRepoStatus checks if the repository has uncommitted changes
func CheckRepoStatus(dir string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}

	// If output is empty, there are no changes
	return len(output) > 0, nil
} 