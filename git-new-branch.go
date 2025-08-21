package main

import (
	"fmt"
	"os"

	"git-tools/common"
)

type newBranchOptions struct {
	name     string
	checkout bool
	remote   string
}

func main() {
	if !common.IsGitRepository() {
		fmt.Fprintf(os.Stderr, "%sError: This directory is not a git repository.%s\n", common.ColorRed, common.ColorReset)
		os.Exit(1)
	}

	opts, err := parseArgs()

	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError: %v%s\n", common.ColorRed, err, common.ColorReset)
		os.Exit(1)
	}

	name, err := common.GetRemoteMainBranch(opts.remote)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError: %v%s\n", common.ColorRed, err, common.ColorReset)
		os.Exit(1)
	}

	mainBranch := fmt.Sprintf("%s/%s", opts.remote, name)
	fmt.Printf("%sFetching '%s'%s\n", common.ColorGreen, mainBranch, common.ColorReset)
	err = common.FetchBranch(opts.remote, name, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError fetching origin branch: %v%s\n", common.ColorRed, err, common.ColorReset)
		os.Exit(1)
	}

	fmt.Printf("%sCreating branch '%s' from '%s'\n", common.ColorGreen, opts.name, mainBranch)
	err = common.CreateBranch(opts.name, mainBranch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError creating branch: %v%s\n", common.ColorRed, err, common.ColorReset)
		os.Exit(1)
	}

	if opts.checkout {
		fmt.Printf("%sChecking out branch '%s'\n", common.ColorGreen, opts.name)
		if err := common.Checkout(opts.name); err != nil {
			fmt.Fprintf(os.Stderr, "%sError checking out branch: %v%s\n", common.ColorRed, err, common.ColorReset)
			os.Exit(1)
		}
	}
	fmt.Printf("%s✅ Branch '%s' created successfully.%s\n", common.ColorGreen, opts.name, common.ColorReset)
}

func parseArgs() (*newBranchOptions, error) {
	opts := &newBranchOptions{
		remote:   "origin",
		checkout: true,
	}
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	var name string = ""

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--remote", "-r":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing argument for %s", arg)
			}
			opts.remote = args[i+1]
			i++
		case "--no-checkout", "-n":
			opts.checkout = false
		default:
			if name != "" {
				return nil, fmt.Errorf("unknown argument: %s", arg)
			}
			name = arg
		}
	}

	if name == "" {
		return nil, fmt.Errorf("missing branch name")
	}
	opts.name = name

	return opts, nil
}

func printUsage() {
	fmt.Println("Usage: git-new-branch [options] <branch name>")
	fmt.Println("Options:")
	fmt.Println("  --remote, -r      Specify the remote name (default: origin)")
	fmt.Println("  --no-checkout, -n  Do not check out the new branch")
	fmt.Println("  --help, -h        Show this help message")
}
