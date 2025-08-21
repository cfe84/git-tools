package main

import (
	"fmt"
	"os"

	"git-tools/common"
)

type getOptions struct {
	subcommand    string
	remote        string
	includeRemote bool
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

	if opts.subcommand == "main-branch" {
		name, err := common.GetRemoteMainBranch(opts.remote)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError: %v%s\n", common.ColorRed, err, common.ColorReset)
			os.Exit(1)
		}

		if opts.includeRemote {
			fmt.Printf("%s/", opts.remote)
		}
		fmt.Println(name)
	}
}

func parseArgs() (*getOptions, error) {
	opts := &getOptions{
		remote:        "origin",
		includeRemote: false,
	}
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	// First argument is either help, or a subcommand.
	if args[0] == "--help" || args[0] == "-h" {
		printUsage()
		os.Exit(0)
	}

	// There's only one subcommand for now.
	if args[0] != "main-branch" {
		return nil, fmt.Errorf("unknown subcommand: %s", args[0])
	}

	opts.subcommand = args[0]
	args = args[1:]

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--remote", "-r":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing argument for %s", arg)
			}
			opts.remote = args[i+1]
			i++
		case "--include-remote", "-i":
			opts.includeRemote = true
		default:
			return nil, fmt.Errorf("unknown argument: %s", arg)
		}

	}

	return opts, nil
}

func printUsage() {
	fmt.Println("Usage: git-get [subcommand] [options]")
	fmt.Println("Subcommands:")
	fmt.Println("  main-branch       Get the main branch name from the remote")
	fmt.Println("Options:")
	fmt.Println("  --remote, -r      Specify the remote name (default: origin)")
	fmt.Println("  --include-remote, -i Include the remote name in the output")
	fmt.Println("  --help, -h        Show this help message")
}
