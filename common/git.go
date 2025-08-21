package common

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// isGitRepository checks if the current directory is a git repository
func IsGitRepository() bool {
	if _, err := os.Stat(".git"); err == nil {
		return true
	}

	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// getGitDirectory returns the path to the .git directory
func GetGitDirectory() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// gitRefExists checks if a git reference exists
func GitRefExists(ref string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", ref)
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// getBranchName tries to get the branch name from a git reference
func GetBranchName(ref string) string {
	cmd := exec.Command("git", "symbolic-ref", "--short", ref)
	cmd.Stderr = nil
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getCurrentBranch gets the current branch name
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "", fmt.Errorf("not on a branch (detached HEAD)")
	}
	return branch, nil
}

// createBranch creates a new git branch from the specified reference
func CreateBranch(branchName, fromRef string) error {
	cmd := exec.Command("git", "branch", branchName, fromRef)
	return cmd.Run()
}

// runGitBackup runs the git backup command
func RunGitBackup() error {
	cmd := exec.Command("git-backup")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runGitBackupWithRef runs the git backup command for the specified reference
func RunGitBackupWithRef(ref string) error {
	cmd := exec.Command("git-backup", ref)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// getCommitHash gets the commit hash for a given reference
func GetCommitHash(ref string) (string, error) {
	cmd := exec.Command("git", "rev-parse", ref)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func Checkout(commit string) error {
	cmd := exec.Command("git", "checkout", commit)
	return cmd.Run()
}

// moveBranch moves a branch to point to a new reference
func MoveBranch(branchName, newRef string) error {
	cmd := exec.Command("git", "branch", "-f", branchName, newRef)
	return cmd.Run()
}

// isCherryPickInProgress checks if a cherry-pick operation is in progress
func IsCherryPickInProgress() bool {
	gitDir, err := GetGitDirectory()
	if err != nil {
		return false
	}

	// Check if CHERRY_PICK_HEAD exists
	cherryPickHead := filepath.Join(gitDir, "CHERRY_PICK_HEAD")
	if _, err := os.Stat(cherryPickHead); err == nil {
		return true
	}

	return false
}

// hasUncommittedChanges checks if there are uncommitted changes
func HasUncommittedChanges() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(output)) != ""
}

// hasUnstagedChanges checks if there are unstaged changes
func HasUnstagedChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if len(line) >= 2 {
			// Check if the working tree status (second character) indicates changes
			workingTreeStatus := line[1]
			if workingTreeStatus == 'M' || workingTreeStatus == 'D' || workingTreeStatus == 'T' {
				return true, nil
			}
			// Check for untracked files (marked as ??)
			if line[0] == '?' && line[1] == '?' {
				return true, nil
			}
		}
	}
	return false, nil
}

// hasStagedChanges checks if there are staged changes
func HasStagedChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if len(line) >= 2 {
			// Check if the index status (first character) indicates staged changes
			indexStatus := line[0]
			if indexStatus == 'M' || indexStatus == 'A' || indexStatus == 'D' ||
				indexStatus == 'R' || indexStatus == 'C' || indexStatus == 'T' {
				return true, nil
			}
		}
	}
	return false, nil
}

// hasConflicts checks if there are merge conflicts
func HasConflicts() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "UU ") || strings.HasPrefix(line, "AA ") ||
			strings.HasPrefix(line, "DD ") || strings.HasPrefix(line, "AU ") ||
			strings.HasPrefix(line, "UD ") || strings.HasPrefix(line, "UA ") ||
			strings.HasPrefix(line, "DU ") || strings.HasPrefix(line, "AD ") {
			return true
		}
	}
	return false
}

// continueCherryPick continues a cherry-pick operation
func ContinueCherryPick() error {
	cmd := exec.Command("git", "cherry-pick", "--continue")
	return cmd.Run()
}

// abortCherryPick aborts a cherry-pick operation
func AbortCherryPick() error {
	cmd := exec.Command("git", "cherry-pick", "--abort")
	return cmd.Run()
}

// cherryPickCommit cherry-picks a specific commit
func CherryPickCommit(commit string) error {
	cmd := exec.Command("git", "cherry-pick", commit)
	return cmd.Run()
}

// getCommitMessage gets the commit message for a given commit
func GetCommitMessage(commit string) (string, error) {
	cmd := exec.Command("git", "log", "--format=%s", "-n", "1", commit)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// createStagedDiff creates a diff file of staged changes
func CreateStagedDiff(filename string) error {
	cmd := exec.Command("git", "diff", "--staged")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	return os.WriteFile(filename, output, 0644)
}

// amendCommit amends the previous commit with staged changes
func AmendCommit() error {
	cmd := exec.Command("git", "commit", "--amend", "--no-edit")
	return cmd.Run()
}

// applyReverseDiff applies a diff file in reverse
func ApplyReverseDiff(filename string) error {
	cmd := exec.Command("git", "apply", "--reverse", filename)
	return cmd.Run()
}

// stageAllChanges stages all changes in the working directory
func StageAllChanges() error {
	cmd := exec.Command("git", "add", "-A")
	return cmd.Run()
}

// createCommit creates a new commit with an optional message
func CreateCommit(message string) error {
	if message != "" {
		cmd := exec.Command("git", "commit", "-m", message)
		return cmd.Run()
	} else {
		cmd := exec.Command("git", "commit")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
}

// deleteBranch deletes a git branch using git branch -D
func DeleteBranch(branchName string) error {
	cmd := exec.Command("git", "branch", "-D", branchName)
	return cmd.Run()
}

// getAllBranches gets all git branches (local and remote)
func GetAllBranches() ([]string, error) {
	cmd := exec.Command("git", "branch", "-a")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	var branches []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Remove the current branch marker (*) and any leading spaces
		line = strings.TrimLeft(line, "* ")

		if line != "" {
			branches = append(branches, line)
		}
	}

	return branches, nil
}

// Get the main branch on a remote
func GetRemoteMainBranch(remote string) (string, error) {
	ref := remote + "/HEAD"
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", ref)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git command failed: %s", strings.TrimSpace(out.String()))
	}

	result := strings.TrimSpace(out.String())
	parts := strings.Split(result, "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("unexpected git output: %q", result)
	}
	return strings.Join(parts[1:], "/"), nil
}

// getCommitRange gets a range of commits using git rev-list
func GetCommitRange(revRange string, reverse bool) ([]string, error) {
	args := []string{"rev-list"}
	if reverse {
		args = append(args, "--reverse")
	}
	args = append(args, revRange)

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	commits := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(commits) == 1 && commits[0] == "" {
		return []string{}, nil
	}
	return commits, nil
}

// isBranch checks if a reference is a local branch
func IsBranch(ref string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+ref)
	return cmd.Run() == nil
}

// writeRefFile writes a commit hash directly to a git ref file
func WriteRefFile(refName, commitHash string) error {
	gitDir, err := GetGitDirectory()
	if err != nil {
		return err
	}

	refPath := filepath.Join(gitDir, "refs", "heads", refName)

	// Create the refs/heads directory if it doesn't exist
	refsHeadsDir := filepath.Dir(refPath)
	if err := os.MkdirAll(refsHeadsDir, 0755); err != nil {
		return fmt.Errorf("failed to create refs directory: %v", err)
	}

	// Write the commit hash to the ref file
	if err := os.WriteFile(refPath, []byte(commitHash+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write ref file: %v", err)
	}

	return nil
}
