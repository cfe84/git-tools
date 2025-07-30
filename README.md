A set of additional git command wrappers to help you make crazy stuff in a very unsafe way.

I've been using these for a little bit, but of course they come with no guarantees, use at your own risk.

# Use

This repo contains the following commands:

`git backup`, which makes a backup of your current branch (it basically just creates a new branch with today's date to points to your HEAD). I can't recommend enough to use this before you use the others, just in case.

`git move-branch`, which changes the commit to where a branch is pointing. This is very useful if you work on cumulative changes (e.g. branch 2 is created on top of branch 1) and use something like [`git revise`](https://git-revise.readthedocs.io/en/latest/index.html) to alter the content of branch 1's commit.

`git reparent`, which re-applies commits on top of a new parent. This is very useful when you work on a repository when the main branch keeps diverging, and `git rebase` gives you tons of garbage conflicts.

`git split`, which allows you to selectively delete stuff from the last commit, and apply it to a new commit. This is useful when you want to split commits in a finer grain than what `git revise --cut` would allow you. You start deleting the code that you want in the new commit, stage that, then call `git split`, and that will amend the current commit, then re-apply the change and stage them (optionally committing them directly.)

`git bookmark`, which allows you to create relative bookmarks to git references. Unlike branches, bookmarks store relative references (like `HEAD~2`) and resolve them dynamically when used. This is useful for temporarily marking specific commits relative to your current position. You can create, list, show, checkout bookmarks, and sync branches to bookmark positions.

All these commands contain a `--help` subcommand that displays their usage.

# Install

At least, you need to have the go compiler, and make is a good + (on Mac and Linux it should be there already. On Windows, go `format C:\ && wget https:\\ubuntu.com\latest && .\ubuntu.exe`, or `choco install make`).

Then build using `make all`, and add the `bin` folder to your PATH.

# Note

All of these are just wrappers around git commands. Some of them like `git move-branch` could very easily have been directly interacting with the `.git` folder, but I could not be bothered. Maybe in the future? Probably not.