package main

import (
	"fmt"
	"os"

	"git-tools/common"
)

func main() {
	if !common.IsGitRepository() {
		fmt.Fprintf(os.Stderr, "%sError: This directory is not a git repository.%s\n", common.ColorRed, common.ColorReset)
		os.Exit(1)
	}

	var branchToMove, newReference string
	var shouldBackup, shouldCheckout bool

	// Parse command line arguments
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "--backup" {
			shouldBackup = true
		} else if arg == "--checkout" {
			shouldCheckout = true
		} else if arg == "--help" || arg == "-h" {
			printUsage()
			os.Exit(0)
		} else if arg == "-b" || arg == "--branch" {
			if i+1 >= len(os.Args) {
				fmt.Fprintf(os.Stderr, "%sError: %s requires a branch name%s\n", common.ColorRed, arg, common.ColorReset)
				os.Exit(1)
			}
			i++
			branchToMove = os.Args[i]
		} else if arg == "-t" || arg == "--to" {
			if i+1 >= len(os.Args) {
				fmt.Fprintf(os.Stderr, "%sError: %s requires a reference%s\n", common.ColorRed, arg, common.ColorReset)
				os.Exit(1)
			}
			i++
			newReference = os.Args[i]
		} else {
			fmt.Fprintf(os.Stderr, "%sError: Unknown argument '%s'%s\n", common.ColorRed, arg, common.ColorReset)
			printUsage()
			os.Exit(1)
		}
	}

	// Validate arguments
	if branchToMove == "" {
		fmt.Fprintf(os.Stderr, "%sError: Branch name is required. Use -b or --branch to specify the branch to move.%s\n", common.ColorRed, common.ColorReset)
		printUsage()
		os.Exit(1)
	}

	// Validate that the branch exists
	if !common.GitRefExists(branchToMove) {
		fmt.Fprintf(os.Stderr, "%sError: Branch '%s' does not exist.%s\n", common.ColorRed, branchToMove, common.ColorReset)
		os.Exit(1)
	}

	// Determine the new reference
	if newReference != "" {
		// Validate that the new reference exists
		if !common.GitRefExists(newReference) {
			fmt.Fprintf(os.Stderr, "%sError: Git reference '%s' does not exist.%s\n", common.ColorRed, newReference, common.ColorReset)
			os.Exit(1)
		}
	} else {
		// If no new reference specified, use HEAD
		newReference = "HEAD"
		fmt.Printf("%sNo new reference specified, using HEAD%s\n", common.ColorYellow, common.ColorReset)
	}

	fmt.Printf("%sBranch to move: %s%s\n", common.ColorGreen, branchToMove, common.ColorReset)
	fmt.Printf("%sNew reference:  %s%s\n", common.ColorGreen, newReference, common.ColorReset)

	// Create backup if requested
	if shouldBackup {
		fmt.Printf("%s▶️ Creating backup before moving branch...%s\n", common.ColorYellow, common.ColorReset)
		if err := common.RunGitBackupWithRef(branchToMove); err != nil {
			fmt.Fprintf(os.Stderr, "%s❌ Failed to create backup: %s%s\n", common.ColorRed, err, common.ColorReset)
			os.Exit(1)
		}
		fmt.Printf("%s✅ Backup created successfully%s\n", common.ColorGreen, common.ColorReset)
		fmt.Println()
	}

	// Get current commit of the branch before moving
	oldCommit, err := common.GetCommitHash(branchToMove)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sWarning: Could not get current commit of branch: %s%s\n", common.ColorYellow, err, common.ColorReset)
		oldCommit = "unknown"
	}

	// Get commit hash of the new reference
	newCommit, err := common.GetCommitHash(newReference)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError: Could not get commit hash of new reference: %s%s\n", common.ColorRed, err, common.ColorReset)
		os.Exit(1)
	}

	// Check if the branch to move is the current branch
	currentBranch, err := common.GetCurrentBranch()
	isCurrentBranch := (err == nil && currentBranch == branchToMove)

	// If moving the current branch, checkout the target commit first
	if isCurrentBranch {
		fmt.Printf("%s▶️ Branch '%s' is currently checked out, switching to target commit first...%s\n", common.ColorYellow, branchToMove, common.ColorReset)
		if err := common.CheckoutCommit(newCommit); err != nil {
			fmt.Fprintf(os.Stderr, "%s❌ Failed to checkout target commit: %s%s\n", common.ColorRed, err, common.ColorReset)
			os.Exit(1)
		}
	}

	// Move the branch
	fmt.Printf("%s▶️ Moving branch '%s' to '%s'...%s\n", common.ColorYellow, branchToMove, newReference, common.ColorReset)
	if err := common.MoveBranch(branchToMove, newReference); err != nil {
		fmt.Fprintf(os.Stderr, "%s❌ Failed to move branch: %s%s\n", common.ColorRed, err, common.ColorReset)
		os.Exit(1)
	}

	// Check out the branch if requested or if it was the current branch
	if shouldCheckout || isCurrentBranch {
		fmt.Printf("%s▶️ Checking out branch '%s'...%s\n", common.ColorYellow, branchToMove, common.ColorReset)
		if err := common.CheckoutBranch(branchToMove); err != nil {
			fmt.Fprintf(os.Stderr, "%s❌ Failed to checkout branch after move: %s%s\n", common.ColorRed, err, common.ColorReset)
			fmt.Fprintf(os.Stderr, "%sWarning: Branch was moved successfully, but you may need to manually checkout '%s'%s\n", common.ColorYellow, branchToMove, common.ColorReset)
		}
	}

	fmt.Printf("%s✅ Branch '%s' moved successfully!%s\n", common.ColorGreen, branchToMove, common.ColorReset)

	// Show summary
	fmt.Println()
	fmt.Printf("%sMove Summary:%s\n", common.ColorCyan, common.ColorReset)
	fmt.Printf("%s  Branch:       %s%s\n", common.ColorWhite, branchToMove, common.ColorReset)
	fmt.Printf("%s  From commit:  %s%s\n", common.ColorWhite, oldCommit[:min(8, len(oldCommit))], common.ColorReset)
	fmt.Printf("%s  To commit:    %s%s\n", common.ColorWhite, newCommit[:min(8, len(newCommit))], common.ColorReset)
	fmt.Printf("%s  Reference:    %s%s\n", common.ColorWhite, newReference, common.ColorReset)
	if shouldBackup {
		fmt.Printf("%s  Backup:       Created%s\n", common.ColorWhite, common.ColorReset)
	}
	if shouldCheckout || isCurrentBranch {
		fmt.Printf("%s  Checked out:  Yes%s\n", common.ColorWhite, common.ColorReset)
	}
}

func printUsage() {
	fmt.Println("git-move-branch - Move a git branch to point to a different commit")
	fmt.Println()
	fmt.Println("Usage: git-move-branch [options] -b <branch-to-move> [-t <new-reference>]")
	fmt.Println()
	fmt.Println("Required Arguments:")
	fmt.Println("  -b, --branch <name>   The name of the branch to move")
	fmt.Println()
	fmt.Println("Optional Arguments:")
	fmt.Println("  -t, --to <reference>  The commit/reference to move the branch to (default: HEAD)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --backup              Create a backup before moving the branch")
	fmt.Println("  --checkout            Check out the branch after moving it")
	fmt.Println("  -h, --help            Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  git-move-branch -b feature-branch                    # Move feature-branch to HEAD")
	fmt.Println("  git-move-branch -b feature-branch -t main            # Move feature-branch to main")
	fmt.Println("  git-move-branch --branch feature-branch --to abc123  # Move feature-branch to commit abc123")
	fmt.Println("  git-move-branch --backup -b feature-branch -t origin/main  # Move with backup")
	fmt.Println("  git-move-branch --checkout -b feature-branch -t main # Move and checkout the branch")
	fmt.Println()
	fmt.Println("Notes:")
	fmt.Println("  - If the branch to move is currently checked out, it will be temporarily")
	fmt.Println("    switched to the target commit before moving, then checked out again")
	fmt.Println("  - Use --backup to create a backup before moving (requires git-backup)")
	fmt.Println("  - The new reference can be any valid git reference (branch, tag, commit hash)")
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
