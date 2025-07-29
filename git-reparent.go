package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"git-tools/common"
)

type reparentOptions struct {
	parentRef     string
	numberOfCommits int
	fromRef       string
	shouldBackup  bool
	shouldConfirm bool
	noBranch      bool
	continueRebase bool
}

func main() {
	if !common.IsGitRepository() {
		fmt.Fprintf(os.Stderr, "%sError: This directory is not a git repository.%s\n", common.ColorRed, common.ColorReset)
		os.Exit(1)
	}

	if len(os.Args) > 1 && os.Args[1] == "--continue" {
		handleContinue()
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "--abort" {
		handleAbort()
		return
	}

	opts, err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError: %s%s\n", common.ColorRed, err, common.ColorReset)
		printUsage()
		os.Exit(1)
	}

	if err := runReparent(opts); err != nil {
		fmt.Fprintf(os.Stderr, "%sError: %s%s\n", common.ColorRed, err, common.ColorReset)
		os.Exit(1)
	}
}

func parseArgs() (*reparentOptions, error) {
	opts := &reparentOptions{
		numberOfCommits: 1, // Default to last commit only
	}

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--parent", "-p":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--parent requires a value")
			}
			opts.parentRef = args[i+1]
			i++
		case "--number", "-n":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--number requires a value")
			}
			num, err := strconv.Atoi(args[i+1])
			if err != nil || num < 1 {
				return nil, fmt.Errorf("--number must be a positive integer")
			}
			opts.numberOfCommits = num
			i++
		case "--from":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--from requires a value")
			}
			opts.fromRef = args[i+1]
			i++
		case "--backup":
			opts.shouldBackup = true
		case "--confirm":
			opts.shouldConfirm = true
		case "--no-branch":
			opts.noBranch = true
		case "--help", "-h":
			printUsage()
			os.Exit(0)
		default:
			return nil, fmt.Errorf("unknown option: %s", arg)
		}
	}

	if opts.parentRef == "" {
		return nil, fmt.Errorf("--parent is required")
	}

	// Validate that both --number and --from are not specified
	if opts.fromRef != "" && opts.numberOfCommits != 1 {
		return nil, fmt.Errorf("cannot specify both --number and --from")
	}

	return opts, nil
}

func runReparent(opts *reparentOptions) error {
	fmt.Printf("%s🔄 Git Reparent Process Starting...%s\n", common.ColorCyan, common.ColorReset)

	if common.HasUncommittedChanges() {
		return fmt.Errorf("there are uncommitted changes. Please commit or stash them first")
	}

	if !common.GitRefExists(opts.parentRef) {
		return fmt.Errorf("parent reference '%s' does not exist", opts.parentRef)
	}

	if opts.shouldBackup {
		fmt.Printf("%s▶️ Creating backup...%s\n", common.ColorYellow, common.ColorReset)
		if err := common.RunGitBackup(); err != nil {
			return fmt.Errorf("failed to create backup: %v", err)
		}
		fmt.Printf("%s✅ Backup created successfully%s\n", common.ColorGreen, common.ColorReset)
	}

	// Get the commit hash of the new parent
	parentCommit, err := common.GetCommitHash(opts.parentRef)
	if err != nil {
		return fmt.Errorf("failed to get parent commit hash: %v", err)
	}

	currentBranch, err := common.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %v", err)
	}
	commits, err := getCommitsToReparent(opts)
	if err != nil {
		return fmt.Errorf("failed to get commits to reparent: %v", err)
	}

	if len(commits) == 0 {
		return fmt.Errorf("no commits to reparent")
	}

	if opts.shouldConfirm {
		fmt.Printf("\n%sReparent Summary:%s\n", common.ColorCyan, common.ColorReset)
		fmt.Printf("%s  Current branch:  %s%s\n", common.ColorWhite, currentBranch, common.ColorReset)
		fmt.Printf("%s  New parent:      %s (%s)%s\n", common.ColorWhite, opts.parentRef, parentCommit[:8], common.ColorReset)
		fmt.Printf("%s  Commits to move: %d%s\n", common.ColorWhite, len(commits), common.ColorReset)
		for i, commit := range commits {
			commitMsg, _ := common.GetCommitMessage(commit)
			fmt.Printf("%s    %d. %s - %s%s\n", common.ColorWhite, i+1, commit[:8], commitMsg, common.ColorReset)
		}
		if !opts.noBranch {
			fmt.Printf("%s  Branch will be moved to new location%s\n", common.ColorWhite, common.ColorReset)
		}

		fmt.Printf("\n%sProceed with reparent? (y/N): %s", common.ColorYellow, common.ColorReset)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Printf("%sReparent cancelled%s\n", common.ColorYellow, common.ColorReset)
			return nil
		}
	}

	fmt.Printf("%s▶️ Checking out new parent as detached HEAD...%s\n", common.ColorYellow, common.ColorReset)
	if err := common.CheckoutCommit(parentCommit); err != nil {
		return fmt.Errorf("failed to checkout parent commit: %v", err)
	}

	if err := saveReparentState(commits, currentBranch, opts.noBranch); err != nil {
		return fmt.Errorf("failed to save reparent state: %v", err)
	}

	if err := applyCherryPicks(commits); err != nil {
		return err
	}

	return finishReparent(currentBranch, opts.noBranch)
}

func handleContinue() {
	fmt.Printf("%s🔄 Continuing git reparent...%s\n", common.ColorCyan, common.ColorReset)

	if !isReparentInProgress() {
		fmt.Fprintf(os.Stderr, "%sError: No reparent in progress%s\n", common.ColorRed, common.ColorReset)
		os.Exit(1)
	}

	state, err := loadReparentState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError: %s%s\n", common.ColorRed, err, common.ColorReset)
		fmt.Fprintf(os.Stderr, "%sUse 'git reparent --abort' to cancel the reparent operation%s\n", common.ColorYellow, common.ColorReset)
		os.Exit(1)
	}

	if common.IsCherryPickInProgress() {
		fmt.Printf("%s▶️ Cherry-pick is in progress, attempting to continue...%s\n", common.ColorYellow, common.ColorReset)
		if err := common.ContinueCherryPick(); err != nil {
			fmt.Fprintf(os.Stderr, "%sError: Failed to continue cherry-pick: %s%s\n", common.ColorRed, err, common.ColorReset)
			fmt.Fprintf(os.Stderr, "%sPlease resolve any remaining conflicts and run 'git cherry-pick --continue' manually%s\n", common.ColorYellow, common.ColorReset)
			os.Exit(1)
		}
		fmt.Printf("%s✅ Cherry-pick continued successfully%s\n", common.ColorGreen, common.ColorReset)
	}

	if err := applyCherryPicks(state.remainingCommits); err != nil {
		fmt.Fprintf(os.Stderr, "%sError: %s%s\n", common.ColorRed, err, common.ColorReset)
		os.Exit(1)
	}

	if err := finishReparent(state.originalBranch, state.noBranch); err != nil {
		fmt.Fprintf(os.Stderr, "%sError: %s%s\n", common.ColorRed, err, common.ColorReset)
		os.Exit(1)
	}
}

func handleAbort() {
	fmt.Printf("%s🔄 Aborting git reparent...%s\n", common.ColorCyan, common.ColorReset)

	if !isReparentInProgress() {
		fmt.Fprintf(os.Stderr, "%sError: No reparent in progress%s\n", common.ColorRed, common.ColorReset)
		os.Exit(1)
	}

	state, err := loadReparentState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError: %s%s\n", common.ColorRed, err, common.ColorReset)
		os.Exit(1)
	}

	// If there's a cherry-pick in progress, abort it first
	if common.IsCherryPickInProgress() {
		fmt.Printf("%s▶️ Aborting cherry-pick in progress...%s\n", common.ColorYellow, common.ColorReset)
		if err := common.AbortCherryPick(); err != nil {
			fmt.Printf("%sWarning: Failed to abort cherry-pick: %v%s\n", common.ColorYellow, err, common.ColorReset)
		}
	}

	fmt.Printf("%s▶️ Checking out original branch '%s'...%s\n", common.ColorYellow, state.originalBranch, common.ColorReset)
	if err := common.CheckoutBranch(state.originalBranch); err != nil {
		fmt.Fprintf(os.Stderr, "%sError: Failed to checkout original branch: %v%s\n", common.ColorRed, err, common.ColorReset)
		os.Exit(1)
	}

	if err := cleanupReparentState(); err != nil {
		fmt.Printf("%sWarning: Failed to cleanup reparent state: %v%s\n", common.ColorYellow, err, common.ColorReset)
	}

	fmt.Printf("%s✅ Reparent aborted successfully%s\n", common.ColorGreen, common.ColorReset)
}

func applyCherryPicks(commits []string) error {
	for i, commit := range commits {
		fmt.Printf("%s▶️ Cherry-picking commit %d/%d: %s%s\n", common.ColorYellow, i+1, len(commits), commit[:8], common.ColorReset)
		
		if err := common.CherryPickCommit(commit); err != nil {
			if common.HasConflicts() {
				fmt.Printf("%s⚠️ Cherry-pick resulted in conflicts%s\n", common.ColorYellow, common.ColorReset)
				fmt.Printf("%sResolve the conflicts and run:%s\n", common.ColorWhite, common.ColorReset)
				fmt.Printf("%s  git add <resolved-files>%s\n", common.ColorWhite, common.ColorReset)
				fmt.Printf("%s  git cherry-pick --continue%s\n", common.ColorWhite, common.ColorReset)
				fmt.Printf("%s  git reparent --continue%s\n", common.ColorWhite, common.ColorReset)
				
				remainingCommits := commits[i+1:]
				if err := updateReparentState(remainingCommits); err != nil {
					return fmt.Errorf("failed to update reparent state: %v", err)
				}
				return fmt.Errorf("cherry-pick conflicts require manual resolution")
			}
			return fmt.Errorf("cherry-pick failed: %v", err)
		}
		fmt.Printf("%s✅ Cherry-pick successful%s\n", common.ColorGreen, common.ColorReset)
	}
	return nil
}

func finishReparent(originalBranch string, noBranch bool) error {
	// Get the current HEAD commit (where we are after cherry-picks)
	newHead, err := common.GetCommitHash("HEAD")
	if err != nil {
		return fmt.Errorf("failed to get new HEAD: %v", err)
	}

	if err := cleanupReparentState(); err != nil {
		fmt.Printf("%sWarning: Failed to cleanup reparent state: %v%s\n", common.ColorYellow, err, common.ColorReset)
	}

	if !noBranch {
		fmt.Printf("%s▶️ Moving branch '%s' to new location...%s\n", common.ColorYellow, originalBranch, common.ColorReset)
		if err := common.MoveBranch(originalBranch, newHead); err != nil {
			return fmt.Errorf("failed to move branch: %v", err)
		}

		fmt.Printf("%s▶️ Checking out branch '%s'...%s\n", common.ColorYellow, originalBranch, common.ColorReset)
		if err := common.CheckoutBranch(originalBranch); err != nil {
			return fmt.Errorf("failed to checkout branch: %v", err)
		}
	}

	fmt.Printf("%s🎉 Reparent completed successfully!%s\n", common.ColorGreen, common.ColorReset)
	return nil
}

func getCommitsToReparent(opts *reparentOptions) ([]string, error) {
	var revRange string
	
	if opts.fromRef != "" {
		// Get commits from fromRef to HEAD
		if !common.GitRefExists(opts.fromRef) {
			return nil, fmt.Errorf("from reference '%s' does not exist", opts.fromRef)
		}
		revRange = fmt.Sprintf("%s..HEAD", opts.fromRef)
	} else {
		// Get last N commits
		revRange = fmt.Sprintf("HEAD~%d..HEAD", opts.numberOfCommits)
	}
	
	return common.GetCommitRange(revRange, true)
}

type reparentState struct {
	remainingCommits []string
	originalBranch   string
	noBranch         bool
}

func getReparentStateFile() (string, error) {
	gitDir, err := common.GetGitDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(gitDir, "git-reparent-state"), nil
}

func saveReparentState(commits []string, originalBranch string, noBranch bool) error {
	stateFile, err := getReparentStateFile()
	if err != nil {
		return err
	}
	
	content := fmt.Sprintf("ORIGINAL_BRANCH=%s\n", originalBranch)
	content += fmt.Sprintf("NO_BRANCH=%t\n", noBranch)
	content += "COMMITS=\n"
	for _, commit := range commits {
		content += fmt.Sprintf("%s\n", commit)
	}
	
	if err := os.WriteFile(stateFile, []byte(content), 0644); err != nil {
		return err
	}
	
	return createReparentHead()
}

func loadReparentState() (*reparentState, error) {
	stateFile, err := getReparentStateFile()
	if err != nil {
		return nil, err
	}
	
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("no reparent in progress")
	}
	
	content, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, err
	}
	
	lines := strings.Split(string(content), "\n")
	state := &reparentState{}
	
	inCommits := false
	for _, line := range lines {
		if strings.HasPrefix(line, "ORIGINAL_BRANCH=") {
			state.originalBranch = strings.TrimPrefix(line, "ORIGINAL_BRANCH=")
		} else if strings.HasPrefix(line, "NO_BRANCH=") {
			state.noBranch = strings.TrimPrefix(line, "NO_BRANCH=") == "true"
		} else if line == "COMMITS=" {
			inCommits = true
		} else if inCommits && line != "" {
			state.remainingCommits = append(state.remainingCommits, line)
		}
	}
	
	return state, nil
}

func updateReparentState(remainingCommits []string) error {
	state, err := loadReparentState()
	if err != nil {
		return err
	}
	
	state.remainingCommits = remainingCommits
	return saveReparentState(state.remainingCommits, state.originalBranch, state.noBranch)
}

func cleanupReparentState() error {
	stateFile, err := getReparentStateFile()
	if err != nil {
		return err
	}
	
	if _, err := os.Stat(stateFile); err == nil {
		if err := os.Remove(stateFile); err != nil {
			return err
		}
	}
	
	return removeReparentHead()
}

func createReparentHead() error {
	gitDir, err := common.GetGitDirectory()
	if err != nil {
		return err
	}
	
	headCommit, err := common.GetCommitHash("HEAD")
	if err != nil {
		return err
	}
	
	reparentHeadFile := filepath.Join(gitDir, "REPARENT_HEAD")
	return os.WriteFile(reparentHeadFile, []byte(headCommit+"\n"), 0644)
}

func removeReparentHead() error {
	gitDir, err := common.GetGitDirectory()
	if err != nil {
		return err
	}
	
	reparentHeadFile := filepath.Join(gitDir, "REPARENT_HEAD")
	if _, err := os.Stat(reparentHeadFile); os.IsNotExist(err) {
		return nil // Already removed
	}
	
	return os.Remove(reparentHeadFile)
}

func isReparentInProgress() bool {
	gitDir, err := common.GetGitDirectory()
	if err != nil {
		return false
	}
	
	reparentHeadFile := filepath.Join(gitDir, "REPARENT_HEAD")
	if _, err := os.Stat(reparentHeadFile); err == nil {
		return true
	}
	
	return false
}

func printUsage() {
	fmt.Println("git reparent - Reparent commits to a new parent. This is useful when histories diverges and git rebase")
	fmt.Println("generates too many conflicts.")
	fmt.Println()
	fmt.Println("Usage: git reparent [options]")
	fmt.Println("       git reparent --continue")
	fmt.Println("       git reparent --abort")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -p, --parent <ref>    New parent reference (required)")
	fmt.Println("  -n, --number <num>    Number of commits to reparent (default: 1)")
	fmt.Println("      --from <ref>      Reparent all commits from <ref> to HEAD")
	fmt.Println("      --backup          Create a backup before reparenting")
	fmt.Println("      --confirm         Show summary and ask for confirmation")
	fmt.Println("      --no-branch       Don't move the branch, leave it detached")
	fmt.Println("      --continue        Continue after resolving conflicts")
	fmt.Println("      --abort           Abort the reparent and return to original branch")
	fmt.Println("  -h, --help            Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  git reparent -p origin/main                    # Reparent last commit to origin/main")
	fmt.Println("  git reparent -p main -n 3                      # Reparent last 3 commits to main")
	fmt.Println("  git reparent -p feature-branch --from v1.0     # Reparent all commits since v1.0 to feature-branch")
	fmt.Println("  git reparent -p main --backup --confirm        # Reparent with backup and confirmation")
}
