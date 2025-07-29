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

	var shouldBackup, shouldForce, shouldCommit, shouldNoAdd bool
	var commitMessage string

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "-b", "--backup":
			shouldBackup = true
		case "-f", "--force":
			shouldForce = true
		case "--no-add":
			shouldNoAdd = true
		case "-c", "--commit":
			shouldCommit = true
		case "-m", "--message":
			if i+1 < len(os.Args) {
				i++
				commitMessage = os.Args[i]
				shouldCommit = true // Automatically enable commit if message is provided
			} else {
				fmt.Fprintf(os.Stderr, "%sError: --message requires a value%s\n", common.ColorRed, common.ColorReset)
				os.Exit(1)
			}
		case "--help", "-h":
			printUsage()
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "%sError: Unknown argument '%s'%s\n", common.ColorRed, arg, common.ColorReset)
			printUsage()
			os.Exit(1)
		}
	}

	// Check for parameter incompatibilities
	if shouldNoAdd && shouldCommit {
		fmt.Fprintf(os.Stderr, "%sError: --no-add is incompatible with --commit and --message%s\n", common.ColorRed, common.ColorReset)
		fmt.Fprintf(os.Stderr, "%s--no-add skips staging changes, but --commit/--message requires staged changes to commit%s\n", common.ColorYellow, common.ColorReset)
		os.Exit(1)
	}

	if shouldForce && shouldCommit {
		fmt.Fprintf(os.Stderr, "%sError: --force is incompatible with --commit and --message%s\n", common.ColorRed, common.ColorReset)
		fmt.Fprintf(os.Stderr, "%s--force implies --no-add, which skips staging changes needed for --commit/--message%s\n", common.ColorYellow, common.ColorReset)
		os.Exit(1)
	}

	// If force is set, automatically set no-add and warn the user
	if shouldForce && !shouldNoAdd {
		shouldNoAdd = true
		fmt.Printf("%sWarning: --force flag automatically enables --no-add to prevent staging unstaged changes%s\n", common.ColorYellow, common.ColorReset)
	}

	if !shouldForce {
		hasUnstaged, err := common.HasUnstagedChanges()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError: Could not check for unstaged changes: %s%s\n", common.ColorRed, err, common.ColorReset)
			os.Exit(1)
		}
		if hasUnstaged {
			fmt.Fprintf(os.Stderr, "%sError: There are unstaged changes. Use --force to proceed anyway or stage your changes first.%s\n", common.ColorRed, common.ColorReset)
			os.Exit(1)
		}
	}

	hasStaged, err := common.HasStagedChanges()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError: Could not check for staged changes: %s%s\n", common.ColorRed, err, common.ColorReset)
		os.Exit(1)
	}
	if !hasStaged {
		fmt.Printf("%sNo staged changes found. Nothing to split.%s\n", common.ColorYellow, common.ColorReset)
		os.Exit(0)
	}

	fmt.Printf("%s📝 Git Split Process Starting...%s\n", common.ColorCyan, common.ColorReset)

	if shouldBackup {
		fmt.Printf("%s▶️ Creating backup...%s\n", common.ColorYellow, common.ColorReset)
		if err := common.RunGitBackup(); err != nil {
			fmt.Fprintf(os.Stderr, "%s❌ Failed to create backup: %s%s\n", common.ColorRed, err, common.ColorReset)
			os.Exit(1)
		}
		fmt.Printf("%s✅ Backup created successfully%s\n", common.ColorGreen, common.ColorReset)
	}

	// Create diff file in .git directory
	gitDir, err := common.GetGitDirectory()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError: Could not determine git directory: %s%s\n", common.ColorRed, err, common.ColorReset)
		os.Exit(1)
	}
	diffFile := gitDir + "/git-split.diff"
	fmt.Printf("%s▶️ Creating diff file: %s%s\n", common.ColorYellow, diffFile, common.ColorReset)
	if err := common.CreateStagedDiff(diffFile); err != nil {
		fmt.Fprintf(os.Stderr, "%s❌ Failed to create diff file: %s%s\n", common.ColorRed, err, common.ColorReset)
		os.Exit(1)
	}

	// Ensure cleanup happens even if something fails
	defer func() {
		if err := os.Remove(diffFile); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "%sWarning: Could not remove diff file: %s%s\n", common.ColorYellow, err, common.ColorReset)
		}
	}()

	fmt.Printf("%s▶️ Amending previous commit...%s\n", common.ColorYellow, common.ColorReset)
	if err := common.AmendCommit(); err != nil {
		fmt.Fprintf(os.Stderr, "%s❌ Failed to amend commit: %s%s\n", common.ColorRed, err, common.ColorReset)
		os.Exit(1)
	}
	fmt.Printf("%s✅ Commit amended successfully%s\n", common.ColorGreen, common.ColorReset)

	fmt.Printf("%s▶️ Applying reverse diff to restore working directory...%s\n", common.ColorYellow, common.ColorReset)
	if err := common.ApplyReverseDiff(diffFile); err != nil {
		fmt.Fprintf(os.Stderr, "%s❌ Failed to apply reverse diff: %s%s\n", common.ColorRed, err, common.ColorReset)
		fmt.Fprintf(os.Stderr, "%sWarning: You may need to manually restore your working directory%s\n", common.ColorYellow, common.ColorReset)
		os.Exit(1)
	}
	fmt.Printf("%s✅ Working directory restored%s\n", common.ColorGreen, common.ColorReset)

	if !shouldNoAdd {
		fmt.Printf("%s▶️ Staging all changes...%s\n", common.ColorYellow, common.ColorReset)
		if err := common.StageAllChanges(); err != nil {
			fmt.Fprintf(os.Stderr, "%s❌ Failed to stage changes: %s%s\n", common.ColorRed, err, common.ColorReset)
			os.Exit(1)
		}
		fmt.Printf("%s✅ All changes staged%s\n", common.ColorGreen, common.ColorReset)
	} else {
		fmt.Printf("%s⏭️ Skipping staging changes (--no-add flag set)%s\n", common.ColorYellow, common.ColorReset)
	}

	if shouldCommit {
		fmt.Printf("%s▶️ Creating new commit...%s\n", common.ColorYellow, common.ColorReset)
		if err := common.CreateCommit(commitMessage); err != nil {
			fmt.Fprintf(os.Stderr, "%s❌ Failed to create commit: %s%s\n", common.ColorRed, err, common.ColorReset)
			os.Exit(1)
		}
		fmt.Printf("%s✅ New commit created%s\n", common.ColorGreen, common.ColorReset)
	}

	fmt.Printf("%s🎉 Git split process completed successfully!%s\n", common.ColorGreen, common.ColorReset)
	
	fmt.Println()
	fmt.Printf("%sSplit Summary:%s\n", common.ColorCyan, common.ColorReset)
	fmt.Printf("%s  Previous commit: Amended%s\n", common.ColorWhite, common.ColorReset)
	fmt.Printf("%s  Working dir:     Restored%s\n", common.ColorWhite, common.ColorReset)
	if !shouldNoAdd {
		fmt.Printf("%s  Changes:         Staged%s\n", common.ColorWhite, common.ColorReset)
	} else {
		fmt.Printf("%s  Changes:         Not staged (--no-add)%s\n", common.ColorWhite, common.ColorReset)
	}
	if shouldBackup {
		fmt.Printf("%s  Backup:          Created%s\n", common.ColorWhite, common.ColorReset)
	}
	if shouldCommit {
		if commitMessage != "" {
			fmt.Printf("%s  New commit:      Created with message%s\n", common.ColorWhite, common.ColorReset)
		} else {
			fmt.Printf("%s  New commit:      Created%s\n", common.ColorWhite, common.ColorReset)
		}
	} else {
		fmt.Printf("%s  New commit:      Not created (use --commit to auto-commit)%s\n", common.ColorWhite, common.ColorReset)
	}
}

func printUsage() {
	fmt.Println("git split - Split previous commits by staging staged deletions that you want to split into a new commit.")
	fmt.Println()
	fmt.Println("This is useful when you want to split a commit into smaller parts (e.g. to simplify PR review process).")
	fmt.Println("Start deleting the code you want to split into a new commit, stage it, then run git split. It will:")
	fmt.Println("- amend the previous commit with staged changes")
	fmt.Println("- restore the working directory to its state before the split and stage all changes (optionally ")
	fmt.Println("  create a new commit)")
	fmt.Println()
	fmt.Println("Usage: git split [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --backup              Create a backup before splitting")
	fmt.Println("  --force               Proceed even if there are unstaged changes (implies --no-add)")
	fmt.Println("  --no-add              Skip staging all changes after restoring working directory")
	fmt.Println("  --commit              Create a new commit after restoring changes")
	fmt.Println("  -m, --message <msg>   Commit message for the new commit (implies --commit)")
	fmt.Println("  -h, --help            Show this help message")
}
