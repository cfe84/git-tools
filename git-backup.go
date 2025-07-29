package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"git-tools/common"
)

func main() {
	if !common.IsGitRepository() {
		fmt.Fprintf(os.Stderr, "%sError: This directory is not a git repository.%s\n", common.ColorRed, common.ColorReset)
		os.Exit(1)
	}

	var targetRef, targetBranch string
	var err error
	var purgeMode, forceMode, listMode bool

	var gitRef string
	for i, arg := range os.Args[1:] {
		switch arg {
		case "-h", "--help":
			printUsage()
			os.Exit(0)
		case "--purge":
			purgeMode = true
		case "--force":
			forceMode = true
		case "-l", "--list":
			listMode = true
		default:
			if gitRef == "" && !purgeMode && !listMode {
				gitRef = arg
			} else if gitRef == "" && (purgeMode || listMode) {
				fmt.Fprintf(os.Stderr, "%sError: --purge and --list do not accept a git reference argument%s\n", common.ColorRed, common.ColorReset)
				os.Exit(1)
			} else {
				fmt.Fprintf(os.Stderr, "%sError: Unknown argument '%s'%s\n", common.ColorRed, arg, common.ColorReset)
				printUsage()
				os.Exit(1)
			}
		}
		_ = i // Suppress unused variable warning
	}

	if purgeMode {
		handlePurgeMode(forceMode)
		return
	}

	if listMode {
		handleListMode()
		return
	}

	if gitRef != "" {
		if !common.GitRefExists(gitRef) {
			fmt.Fprintf(os.Stderr, "%sError: Git reference '%s' does not exist.%s\n", common.ColorRed, gitRef, common.ColorReset)
			os.Exit(1)
		}

		branchName := common.GetBranchName(gitRef)
		if branchName != "" {
			targetBranch = branchName
			targetRef = gitRef
		} else {
			// It's not a branch (could be a commit hash, tag, etc.)
			targetBranch = gitRef
			targetRef = gitRef
		}

		fmt.Printf("%sTarget reference: %s%s\n", common.ColorGreen, gitRef, common.ColorReset)
		if targetBranch != gitRef {
			fmt.Printf("%sResolved to branch: %s%s\n", common.ColorGreen, targetBranch, common.ColorReset)
		}
	} else {
		targetBranch, err = common.GetCurrentBranch()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError: Could not determine current branch name: %s%s\n", common.ColorRed, err, common.ColorReset)
			os.Exit(1)
		}
		targetRef = targetBranch
		fmt.Printf("%sCurrent branch: %s%s\n", common.ColorGreen, targetBranch, common.ColorReset)
	}

	if common.HasUncommittedChanges() {
		fmt.Printf("%s⚠️  Warning: You have uncommitted changes in your working directory.%s\n", common.ColorYellow, common.ColorReset)
		fmt.Printf("%s   The backup will capture the current state of the '%s' branch,\n", common.ColorYellow, targetBranch)
		fmt.Printf("   but your uncommitted changes will not be included in the backup.%s\n", common.ColorReset)
		fmt.Println()
	}

	// Get today's date in yyyy-mm-dd format
	dateStr := time.Now().Format("2006-01-02")

	baseBackupName := fmt.Sprintf("backups/%s/%s", targetBranch, dateStr)
	existingBackups := getExistingBackups(baseBackupName)
	backupNumber := getNextBackupNumber(existingBackups, baseBackupName)

	var backupBranchName string
	if backupNumber == 1 && !hasExactMatch(existingBackups, baseBackupName) {
		backupBranchName = baseBackupName
	} else {
		backupBranchName = fmt.Sprintf("%s-%d", baseBackupName, backupNumber)
	}

	fmt.Printf("%s ▶️ Creating backup branch: %s%s\n", common.ColorYellow, backupBranchName, common.ColorReset)

	if err := common.CreateBranch(backupBranchName, targetRef); err != nil {
		fmt.Fprintf(os.Stderr, "%s❌ Failed to create backup branch: %s%s\n", common.ColorRed, err, common.ColorReset)
		os.Exit(1)
	}

	fmt.Printf("%s ✅ Backup branch '%s' created successfully!%s\n", common.ColorGreen, backupBranchName, common.ColorReset)

	fmt.Println()
	fmt.Printf("%sBackup Summary:%s\n", common.ColorCyan, common.ColorReset)
	fmt.Printf("%s  Source reference: %s%s\n", common.ColorWhite, targetRef, common.ColorReset)
	fmt.Printf("%s  Backup branch:    %s%s\n", common.ColorWhite, backupBranchName, common.ColorReset)
}

// getExistingBackups gets all existing backup branches for today
func getExistingBackups(baseBackupName string) []string {
	branches, err := common.GetAllBranches()
	if err != nil {
		return nil
	}

	var backups []string
	
	pattern := fmt.Sprintf(`^\s*%s(-\d+)?$`, regexp.QuoteMeta(baseBackupName))
	regex := regexp.MustCompile(pattern)

	for _, branch := range branches {
		if regex.MatchString(branch) {
			backups = append(backups, branch)
		}
	}

	return backups
}

func getNextBackupNumber(existingBackups []string, baseBackupName string) int {
	if len(existingBackups) == 0 {
		return 1
	}

	var numbers []int
	numberPattern := fmt.Sprintf(`%s-(\d+)`, regexp.QuoteMeta(baseBackupName))
	exactPattern := fmt.Sprintf(`^%s$`, regexp.QuoteMeta(baseBackupName))
	
	numberRegex := regexp.MustCompile(numberPattern)
	exactRegex := regexp.MustCompile(exactPattern)

	for _, backup := range existingBackups {
		if matches := numberRegex.FindStringSubmatch(backup); matches != nil {
			if num, err := strconv.Atoi(matches[1]); err == nil {
				numbers = append(numbers, num)
			}
		} else if exactRegex.MatchString(backup) {
			numbers = append(numbers, 0)
		}
	}

	if len(numbers) == 0 {
		return 1
	}

	sort.Ints(numbers)
	return numbers[len(numbers)-1] + 1
}

func hasExactMatch(existingBackups []string, baseBackupName string) bool {
	pattern := fmt.Sprintf(`^%s$`, regexp.QuoteMeta(baseBackupName))
	regex := regexp.MustCompile(pattern)

	for _, backup := range existingBackups {
		if regex.MatchString(backup) {
			return true
		}
	}
	return false
}

func handlePurgeMode(forceMode bool) {
	currentBranch, err := common.GetCurrentBranch()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError: Could not determine current branch name: %s%s\n", common.ColorRed, err, common.ColorReset)
		os.Exit(1)
	}

	backupPattern := fmt.Sprintf("backups/%s/", currentBranch)
	backupBranches := getAllBackupBranches(backupPattern)

	if len(backupBranches) == 0 {
		fmt.Printf("%sNo backup branches found for branch '%s'%s\n", common.ColorYellow, currentBranch, common.ColorReset)
		return
	}

	fmt.Printf("%sFound %d backup branch(es) for '%s':%s\n", common.ColorCyan, len(backupBranches), currentBranch, common.ColorReset)
	for _, branch := range backupBranches {
		fmt.Printf("%s  - %s%s\n", common.ColorWhite, branch, common.ColorReset)
	}
	fmt.Println()

	if !forceMode {
		fmt.Printf("%sAre you sure you want to delete all %d backup branches for '%s'? [y/N]: %s", 
			common.ColorYellow, len(backupBranches), currentBranch, common.ColorReset)
		
		var response string
		fmt.Scanln(&response)
		
		if response != "y" && response != "Y" && response != "yes" && response != "YES" {
			fmt.Printf("%sPurge operation cancelled%s\n", common.ColorYellow, common.ColorReset)
			return
		}
	}

	fmt.Printf("%s▶️ Deleting backup branches...%s\n", common.ColorYellow, common.ColorReset)
	
	deletedCount := 0
	for _, branch := range backupBranches {
		if err := common.DeleteBranch(branch); err != nil {
			fmt.Fprintf(os.Stderr, "%s❌ Failed to delete branch '%s': %s%s\n", common.ColorRed, branch, err, common.ColorReset)
		} else {
			fmt.Printf("%s  ✅ Deleted %s%s\n", common.ColorGreen, branch, common.ColorReset)
			deletedCount++
		}
	}

	fmt.Printf("%s🎉 Successfully deleted %d/%d backup branches for '%s'%s\n", 
		common.ColorGreen, deletedCount, len(backupBranches), currentBranch, common.ColorReset)
}

func handleListMode() {
	currentBranch, err := common.GetCurrentBranch()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError: Could not determine current branch name: %s%s\n", common.ColorRed, err, common.ColorReset)
		os.Exit(1)
	}

	backupPattern := fmt.Sprintf("backups/%s/", currentBranch)
	backupBranches := getAllBackupBranches(backupPattern)

	if len(backupBranches) == 0 {
		fmt.Printf("%sNo backup branches found for branch '%s'%s\n", common.ColorYellow, currentBranch, common.ColorReset)
		return
	}

	fmt.Printf("%sBackup branches for '%s':%s\n", common.ColorCyan, currentBranch, common.ColorReset)
	
	sort.Strings(backupBranches)
	
	for i, branch := range backupBranches {
		commitHash, err := common.GetCommitHash(branch)
		if err != nil {
			fmt.Printf("%s  %d. %s %s(commit unknown)%s\n", common.ColorWhite, i+1, branch, common.ColorYellow, common.ColorReset)
		} else {
			commitMsg, err := common.GetCommitMessage(branch)
			if err != nil {
				fmt.Printf("%s  %d. %s %s(%s)%s\n", common.ColorWhite, i+1, branch, common.ColorYellow, commitHash[:8], common.ColorReset)
			} else {
				fmt.Printf("%s  %d. %s %s(%s)%s - %s\n", common.ColorWhite, i+1, branch, common.ColorYellow, commitHash[:8], common.ColorReset, commitMsg)
			}
		}
	}
	
	fmt.Printf("\n%sTotal: %d backup(s)%s\n", common.ColorCyan, len(backupBranches), common.ColorReset)
}

func getAllBackupBranches(pattern string) []string {
	branches, err := common.GetAllBranches()
	if err != nil {
		return nil
	}

	var backups []string
	
	for _, branch := range branches {
		if strings.HasPrefix(branch, pattern) {
			backups = append(backups, branch)
		}
	}

	return backups
}

func printUsage() {
	fmt.Println("git-backup - Create a backup branch from a git reference")
	fmt.Println()
	fmt.Println("Usage: git-backup [options] [reference]")
	fmt.Println("       git-backup --purge [--force]")
	fmt.Println("       git-backup --list")
	fmt.Println()
	fmt.Println("Arguments:")
	fmt.Println("  reference    Git reference to backup (branch, commit, tag)")
	fmt.Println("               If not provided, backs up the current branch")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --list, -l   List all backup branches for the current branch")
	fmt.Println("  --purge      Delete all backup branches for the current branch")
	fmt.Println("  --force      Skip confirmation when using --purge")
	fmt.Println("  -h, --help   Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  git-backup                    # Backup current branch")
	fmt.Println("  git-backup main               # Backup the main branch")
	fmt.Println("  git-backup feature/new-ui     # Backup a feature branch")
	fmt.Println("  git-backup abc123             # Backup a specific commit")
	fmt.Println("  git-backup --list             # List all backup branches for current branch")
	fmt.Println("  git-backup --purge            # Delete all backups of current branch (with confirmation)")
	fmt.Println("  git-backup --purge --force    # Delete all backups of current branch (no confirmation)")
	fmt.Println()
	fmt.Println("Backup branches are created under:")
	fmt.Println("  backups/<branch-name>/<date>[-number]")
	fmt.Println()
	fmt.Println("Where:")
	fmt.Println("  <branch-name> is the source branch name")
	fmt.Println("  <date> is today's date (yyyy-mm-dd)")
	fmt.Println("  [-number] is added if multiple backups exist for the same day")
}
