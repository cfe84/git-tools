package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"git-tools/common"
)

type bookmarkOptions struct {
	action      string
	name        string
	reference   string
	absolute    bool
	interactive bool
}

func main() {
	if !common.IsGitRepository() {
		fmt.Fprintf(os.Stderr, "%sError: This directory is not a git repository.%s\n", common.ColorRed, common.ColorReset)
		os.Exit(1)
	}

	opts, err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError: %s%s\n", common.ColorRed, err, common.ColorReset)
		printUsage()
		os.Exit(1)
	}

	switch opts.action {
	case "create":
		if err := createBookmark(opts.name, opts.reference); err != nil {
			fmt.Fprintf(os.Stderr, "%sError: %s%s\n", common.ColorRed, err, common.ColorReset)
			os.Exit(1)
		}
	case "delete":
		if err := deleteBookmark(opts.name); err != nil {
			fmt.Fprintf(os.Stderr, "%sError: %s%s\n", common.ColorRed, err, common.ColorReset)
			os.Exit(1)
		}
	case "show":
		if err := showBookmark(opts.name, opts.absolute); err != nil {
			fmt.Fprintf(os.Stderr, "%sError: %s%s\n", common.ColorRed, err, common.ColorReset)
			os.Exit(1)
		}
	case "list":
		if err := listBookmarks(); err != nil {
			fmt.Fprintf(os.Stderr, "%sError: %s%s\n", common.ColorRed, err, common.ColorReset)
			os.Exit(1)
		}
	case "checkout":
		if err := checkoutBookmark(opts.name); err != nil {
			fmt.Fprintf(os.Stderr, "%sError: %s%s\n", common.ColorRed, err, common.ColorReset)
			os.Exit(1)
		}
	case "checkout-previous":
		if err := checkoutPreviousBookmark(); err != nil {
			fmt.Fprintf(os.Stderr, "%sError: %s%s\n", common.ColorRed, err, common.ColorReset)
			os.Exit(1)
		}
	case "interactive":
		if err := interactiveCheckout(); err != nil {
			fmt.Fprintf(os.Stderr, "%sError: %s%s\n", common.ColorRed, err, common.ColorReset)
			os.Exit(1)
		}
	case "sync":
		if err := syncBranchFromBookmark(opts.name); err != nil {
			fmt.Fprintf(os.Stderr, "%sError: %s%s\n", common.ColorRed, err, common.ColorReset)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "%sError: Unknown action '%s'%s\n", common.ColorRed, opts.action, common.ColorReset)
		printUsage()
		os.Exit(1)
	}
}

func parseArgs() (*bookmarkOptions, error) {
	opts := &bookmarkOptions{}
	args := os.Args[1:]

	if len(args) == 0 {
		return nil, fmt.Errorf("action is required")
	}

	if args[0] == "--help" || args[0] == "-h" {
		printUsage()
		os.Exit(0)
	}

	if args[0] == "-" {
		opts.action = "checkout-previous"
		return opts, nil
	}

	if args[0] == "interactive" {
		opts.action = "interactive"
		return opts, nil
	}

	opts.action = args[0]
	args = args[1:]

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--name", "-n":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("%s requires a value", arg)
			}
			opts.name = args[i+1]
			i++
		case "--absolute", "-a":
			opts.absolute = true
		case "--help", "-h":
			printUsage()
			os.Exit(0)
		default:
			// Handle positional arguments based on action
			if opts.action == "create" {
				if opts.name == "" {
					opts.name = arg
				} else if opts.reference == "" {
					opts.reference = arg
				} else {
					return nil, fmt.Errorf("too many arguments for create action")
				}
			} else if opts.action == "delete" || opts.action == "show" || opts.action == "checkout" || opts.action == "sync" {
				if opts.name == "" {
					opts.name = arg
				} else {
					return nil, fmt.Errorf("too many arguments for %s action", opts.action)
				}
			} else {
				return nil, fmt.Errorf("unknown argument: %s", arg)
			}
		}
	}

	switch opts.action {
	case "create", "delete", "show", "checkout", "sync":
		if opts.name == "" {
			return nil, fmt.Errorf("%s action requires a bookmark name", opts.action)
		}
	case "list":
	default:
		return nil, fmt.Errorf("unknown action: %s", opts.action)
	}

	return opts, nil
}

func getBookmarksDir() (string, error) {
	gitDir, err := common.GetGitDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(gitDir, "bookmarks"), nil
}

func createBookmark(name, reference string) error {
	if reference == "" {
		// Use current branch/HEAD if no reference specified
		currentBranch, err := common.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("current commit is not a branch")
		} else {
			reference = currentBranch
		}
	}

	// Validate that the reference exists (resolve it to ensure it's valid)
	if !common.GitRefExists(reference) {
		return fmt.Errorf("reference '%s' does not exist", reference)
	}

	bookmarksDir, err := getBookmarksDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(bookmarksDir, 0755); err != nil {
		return fmt.Errorf("failed to create bookmarks directory: %v", err)
	}

	bookmarkFile := filepath.Join(bookmarksDir, name)

	if err := os.WriteFile(bookmarkFile, []byte(reference+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to create bookmark: %v", err)
	}

	if err := updatePreviousBookmark(name); err != nil {
		fmt.Printf("%sWarning: Failed to update previous bookmark tracking: %v%s\n", common.ColorYellow, err, common.ColorReset)
	}

	fmt.Printf("%s✅ Bookmark '%s' created pointing to '%s'%s\n", common.ColorGreen, name, reference, common.ColorReset)
	return nil
}

func deleteBookmark(name string) error {
	bookmarksDir, err := getBookmarksDir()
	if err != nil {
		return err
	}

	bookmarkFile := filepath.Join(bookmarksDir, name)

	if _, err := os.Stat(bookmarkFile); os.IsNotExist(err) {
		return fmt.Errorf("bookmark '%s' does not exist", name)
	}

	if err := os.Remove(bookmarkFile); err != nil {
		return fmt.Errorf("failed to delete bookmark: %v", err)
	}

	fmt.Printf("%s✅ Bookmark '%s' deleted%s\n", common.ColorGreen, name, common.ColorReset)
	return nil
}

func showBookmark(name string, absolute bool) error {
	reference, err := getBookmarkReference(name)
	if err != nil {
		return err
	}

	if absolute {
		commitHash, err := common.GetCommitHash(reference)
		if err != nil {
			return fmt.Errorf("failed to resolve bookmark reference: %v", err)
		}
		fmt.Printf("%s%s%s\n", common.ColorGreen, commitHash, common.ColorReset)
	} else {
		fmt.Printf("%s%s%s\n", common.ColorGreen, reference, common.ColorReset)
	}

	return nil
}

func listBookmarks() error {
	bookmarksDir, err := getBookmarksDir()
	if err != nil {
		return err
	}

	if _, err := os.Stat(bookmarksDir); os.IsNotExist(err) {
		fmt.Printf("%sNo bookmarks found%s\n", common.ColorYellow, common.ColorReset)
		return nil
	}

	entries, err := os.ReadDir(bookmarksDir)
	if err != nil {
		return fmt.Errorf("failed to read bookmarks directory: %v", err)
	}

	if len(entries) == 0 {
		fmt.Printf("%sNo bookmarks found%s\n", common.ColorYellow, common.ColorReset)
		return nil
	}

	fmt.Printf("%sBookmarks:%s\n", common.ColorCyan, common.ColorReset)

	var bookmarks []string
	for _, entry := range entries {
		if !entry.IsDir() {
			bookmarks = append(bookmarks, entry.Name())
		}
	}
	sort.Strings(bookmarks)

	for _, name := range bookmarks {
		reference, err := getBookmarkReference(name)
		if err != nil {
			fmt.Printf("%s  %s - %s(error: %v)%s\n", common.ColorWhite, name, common.ColorRed, err, common.ColorReset)
			continue
		}

		commitHash, err := common.GetCommitHash(reference)
		if err != nil {
			fmt.Printf("%s  %s -> %s%s\n", common.ColorWhite, name, reference, common.ColorReset)
		} else {
			fmt.Printf("%s  %s -> %s %s(%s)%s\n", common.ColorWhite, name, reference, common.ColorYellow, commitHash[:8], common.ColorReset)
		}
	}

	return nil
}

func checkoutBookmark(name string) error {
	reference, err := getBookmarkReference(name)
	if err != nil {
		return err
	}

	if err := updatePreviousBookmark(name); err != nil {
		fmt.Printf("%sWarning: Failed to update previous bookmark tracking: %v%s\n", common.ColorYellow, err, common.ColorReset)
	}

	if err := common.Checkout(reference); err != nil {
		return fmt.Errorf("failed to checkout bookmark: %v", err)
	}

	fmt.Printf("%s✅ Checked out bookmark '%s' (%s -> %s)%s\n", common.ColorGreen, name, reference, reference[:8], common.ColorReset)
	return nil
}

func checkoutPreviousBookmark() error {
	previousName, err := getPreviousBookmark()
	if err != nil {
		return err
	}

	if previousName == "" {
		return fmt.Errorf("no previous bookmark to checkout")
	}

	return checkoutBookmark(previousName)
}

func interactiveCheckout() error {
	bookmarksDir, err := getBookmarksDir()
	if err != nil {
		return err
	}

	if _, err := os.Stat(bookmarksDir); os.IsNotExist(err) {
		return fmt.Errorf("no bookmarks found")
	}

	entries, err := os.ReadDir(bookmarksDir)
	if err != nil {
		return fmt.Errorf("failed to read bookmarks directory: %v", err)
	}

	var bookmarks []string
	for _, entry := range entries {
		if !entry.IsDir() {
			bookmarks = append(bookmarks, entry.Name())
		}
	}

	if len(bookmarks) == 0 {
		return fmt.Errorf("no bookmarks found")
	}

	sort.Strings(bookmarks)

	fmt.Printf("%sSelect a bookmark to checkout:%s\n", common.ColorCyan, common.ColorReset)
	for i, name := range bookmarks {
		reference, err := getBookmarkReference(name)
		if err != nil {
			fmt.Printf("%s  %d. %s %s(error)%s\n", common.ColorWhite, i+1, name, common.ColorRed, common.ColorReset)
			continue
		}

		commitHash, err := common.GetCommitHash(reference)
		if err != nil {
			fmt.Printf("%s  %d. %s -> %s%s\n", common.ColorWhite, i+1, name, reference, common.ColorReset)
		} else {
			fmt.Printf("%s  %d. %s -> %s %s(%s)%s\n", common.ColorWhite, i+1, name, reference, common.ColorYellow, commitHash[:8], common.ColorReset)
		}
	}

	fmt.Printf("\n%sEnter bookmark number (1-%d): %s", common.ColorYellow, len(bookmarks), common.ColorReset)
	var choice int
	if _, err := fmt.Scanln(&choice); err != nil {
		return fmt.Errorf("invalid input")
	}

	if choice < 1 || choice > len(bookmarks) {
		return fmt.Errorf("invalid choice: %d", choice)
	}

	selectedBookmark := bookmarks[choice-1]
	return checkoutBookmark(selectedBookmark)
}

func syncBranchFromBookmark(name string) error {
	reference, err := getBookmarkReference(name)
	if err != nil {
		return err
	}

	commitHash, err := common.GetCommitHash(reference)
	if err != nil {
		return fmt.Errorf("failed to resolve bookmark reference: %v", err)
	}

	if err := common.WriteRefFile(name, commitHash); err != nil {
		return fmt.Errorf("failed to sync branch: %v", err)
	}

	branchExisted := common.IsBranch(name)
	if branchExisted {
		fmt.Printf("%s✅ Branch '%s' synced to bookmark commit (%s -> %s)%s\n",
			common.ColorGreen, name, reference, commitHash[:8], common.ColorReset)
	} else {
		fmt.Printf("%s✅ Branch '%s' created and synced to bookmark commit (%s -> %s)%s\n",
			common.ColorGreen, name, reference, commitHash[:8], common.ColorReset)
	}

	return nil
}

func getBookmarkReference(name string) (string, error) {
	bookmarksDir, err := getBookmarksDir()
	if err != nil {
		return "", err
	}

	bookmarkFile := filepath.Join(bookmarksDir, name)

	if _, err := os.Stat(bookmarkFile); os.IsNotExist(err) {
		return "", fmt.Errorf("bookmark '%s' does not exist", name)
	}

	content, err := os.ReadFile(bookmarkFile)
	if err != nil {
		return "", fmt.Errorf("failed to read bookmark: %v", err)
	}

	return strings.TrimSpace(string(content)), nil
}

func updatePreviousBookmark(currentBookmark string) error {
	gitDir, err := common.GetGitDirectory()
	if err != nil {
		return err
	}

	previousFile := filepath.Join(gitDir, "PREVIOUS_BOOKMARK")

	var previousBookmark string
	if content, err := os.ReadFile(previousFile); err == nil {
		previousBookmark = strings.TrimSpace(string(content))
	}

	if previousBookmark != currentBookmark {
		return os.WriteFile(previousFile, []byte(currentBookmark+"\n"), 0644)
	}

	return nil
}

func getPreviousBookmark() (string, error) {
	gitDir, err := common.GetGitDirectory()
	if err != nil {
		return "", err
	}

	previousFile := filepath.Join(gitDir, "PREVIOUS_BOOKMARK")

	if _, err := os.Stat(previousFile); os.IsNotExist(err) {
		return "", nil
	}

	content, err := os.ReadFile(previousFile)
	if err != nil {
		return "", fmt.Errorf("failed to read previous bookmark: %v", err)
	}

	return strings.TrimSpace(string(content)), nil
}

func printUsage() {
	fmt.Println("git-bookmark - Create and manage relative git bookmarks")
	fmt.Println()
	fmt.Println("Usage: git-bookmark <action> [options] [arguments]")
	fmt.Println()
	fmt.Println("Actions:")
	fmt.Println("  create <name> [reference]  Create a bookmark pointing to a reference (default: current branch/HEAD)")
	fmt.Println("  delete <name>              Delete a bookmark")
	fmt.Println("  show <name>                Show what a bookmark points to")
	fmt.Println("  list                       List all bookmarks")
	fmt.Println("  checkout <name>            Checkout a bookmark")
	fmt.Println("  -                          Checkout the previous bookmark")
	fmt.Println("  interactive                Interactive bookmark selection menu")
	fmt.Println("  sync <name>                Create/update branch to point to bookmark's commit")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -n, --name <name>          Specify bookmark name (alternative to positional arg)")
	fmt.Println("  -a, --absolute             Show absolute commit hash instead of reference (for show)")
	fmt.Println("  -h, --help                 Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  git-bookmark create fixes HEAD~2       # Create bookmark 'fixes' pointing to HEAD~2")
	fmt.Println("  git-bookmark create stable main        # Create bookmark 'stable' pointing to main branch")
	fmt.Println("  git-bookmark list                      # List all bookmarks")
	fmt.Println("  git-bookmark checkout fixes            # Checkout the 'fixes' bookmark")
	fmt.Println("  git-bookmark show fixes --absolute     # Show absolute commit hash for 'fixes'")
	fmt.Println("  git-bookmark -                         # Checkout previous bookmark")
	fmt.Println("  git-bookmark interactive               # Interactive bookmark selection")
	fmt.Println("  git-bookmark sync fixes                # Create/update 'fixes' branch to bookmark's commit")
	fmt.Println()
	fmt.Println("Notes:")
	fmt.Println("  - Bookmarks store relative references (e.g., HEAD~2) and resolve them when used")
	fmt.Println("  - Bookmarks are stored in .git/bookmarks/")
	fmt.Println("  - Use 'git-bookmark -' to quickly switch between bookmarks")
	fmt.Println("  - sync creates the branch if it doesn't exist, or updates it if it does")
}
