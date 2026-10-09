// Package main is the entry point for the helm-env CLI tool.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/user/helm-env/internal/commands"
)

// Version is set at build time via ldflags:
//
//	go build -ldflags "-X main.Version=0.1.0"
var Version = "dev"

func main() {
	// Inject version into commands package
	commands.Version = Version

	args := os.Args[1:]

	if len(args) == 0 {
		commands.Help()
		os.Exit(0)
	}

	var err error

	switch args[0] {
	case "help", "--help", "-h":
		commands.Help()

	case "version", "--version", "-v":
		commands.PrintVersion()

	case "init":
		err = commands.Init()

	case "list":
		err = commands.List()

	case "list-remote":
		includePrerelease := false
		for _, arg := range args[1:] {
			switch arg {
			case "-h", "--help":
				commands.ListRemoteHelp()
				os.Exit(0)
			case "--prerelease":
				includePrerelease = true
			}
		}
		err = commands.ListRemote(includePrerelease)

	case "latest":
		includePrerelease := false
		for _, arg := range args[1:] {
			switch arg {
			case "-h", "--help":
				commands.LatestHelp()
				os.Exit(0)
			case "--prerelease":
				includePrerelease = true
			}
		}
		err = commands.Latest(includePrerelease)

	case "install":
		version := ""
		silent := false
		for _, arg := range args[1:] {
			if arg == "-s" || arg == "--silent" {
				silent = true
			} else if arg == "-h" || arg == "--help" {
				commands.InstallHelp()
				os.Exit(0)
			} else if version == "" && !strings.HasPrefix(arg, "-") {
				version = arg
			}
		}
		err = commands.Install(version, silent)

	case "uninstall":
		version := ""
		if len(args) > 1 {
			version = args[1]
		}
		err = commands.Uninstall(version)

	case "shell":
		version := ""
		if len(args) > 1 {
			version = args[1]
		}
		err = commands.Shell(version)

	case "local":
		version := ""
		if len(args) > 1 {
			version = args[1]
		}
		err = commands.Local(version)

	case "global":
		version := ""
		if len(args) > 1 {
			version = args[1]
		}
		err = commands.Global(version)

	case "which":
		explain := false
		for _, arg := range args[1:] {
			switch arg {
			case "--explain":
				explain = true
			case "-h", "--help":
				commands.WhichHelp()
				os.Exit(0)
			}
		}
		err = commands.Which(explain)

	case "upgrade":
		err = commands.Upgrade()

	case "autocompletion":
		for _, arg := range args[1:] {
			if arg == "-h" || arg == "--help" {
				commands.AutocompletionHelp()
				os.Exit(0)
			}
		}
		err = commands.Autocompletion()

	case "status":
		err = commands.Status()

	case "resolve":
		concrete := true
		for _, arg := range args[1:] {
			switch arg {
			case "--concrete":
				concrete = true
			case "-h", "--help":
				commands.ResolveHelp()
				os.Exit(0)
			}
		}
		err = commands.Resolve(concrete)

	case "prune":
		keep := 0
		olderThan := ""
		dryRun := false
		for i := 1; i < len(args); i++ {
			arg := args[i]
			switch {
			case arg == "--dry-run":
				dryRun = true
			case arg == "--keep":
				if i+1 >= len(args) {
					err = fmt.Errorf("--keep requires a value")
				} else {
					i++
					if _, perr := fmt.Sscanf(args[i], "%d", &keep); perr != nil {
						err = fmt.Errorf("invalid --keep value %q: %w", args[i], perr)
					}
				}
			case strings.HasPrefix(arg, "--keep="):
				if _, perr := fmt.Sscanf(strings.TrimPrefix(arg, "--keep="), "%d", &keep); perr != nil {
					err = fmt.Errorf("invalid --keep value: %w", perr)
				}
			case arg == "--older-than":
				if i+1 >= len(args) {
					err = fmt.Errorf("--older-than requires a value")
				} else {
					i++
					olderThan = args[i]
				}
			case strings.HasPrefix(arg, "--older-than="):
				olderThan = strings.TrimPrefix(arg, "--older-than=")
			case arg == "-h" || arg == "--help":
				commands.PruneHelp()
				os.Exit(0)
			default:
				err = fmt.Errorf("unknown flag %q for prune", arg)
			}
			if err != nil {
				break
			}
		}
		if err == nil {
			err = commands.Prune(keep, olderThan, dryRun)
		}

	case "doctor":
		for _, arg := range args[1:] {
			if arg == "-h" || arg == "--help" {
				commands.DoctorHelp()
				os.Exit(0)
			}
		}
		err = commands.Doctor()

	case "exec":
		version := ""
		execArgs := []string{}
		auto := commands.AutoFromEnv()
		foundVersion := false
		for i := 1; i < len(args); i++ {
			arg := args[i]
			switch arg {
			case "--auto":
				auto = commands.AutoForce
			case "--no-auto":
				auto = commands.AutoDisabled
			default:
				if !foundVersion && !strings.HasPrefix(arg, "-") {
					version = arg
					foundVersion = true
					if i+1 < len(args) {
						execArgs = args[i+1:]
					}
					i = len(args)
				}
			}
		}
		err = commands.ExecWithOptions(version, execArgs, auto)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", args[0])
		commands.Help()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
