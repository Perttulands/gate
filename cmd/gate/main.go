package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"polis/gate/internal/bead"
	"polis/gate/internal/pipeline"
	"polis/gate/internal/verdict"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		printUsage()
		return 0
	}

	if args[0] != "check" {
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		printUsage()
		return 1
	}

	// Parse check arguments
	var repoPath, level, citizen string
	var jsonOutput bool

	level = pipeline.LevelStandard
	i := 1
	for i < len(args) {
		switch args[i] {
		case "--level":
			i++
			if i >= len(args) {
				fmt.Fprintln(os.Stderr, "--level requires a value")
				return 1
			}
			level = args[i]
		case "--json":
			jsonOutput = true
		case "--citizen":
			i++
			if i >= len(args) {
				fmt.Fprintln(os.Stderr, "--citizen requires a value")
				return 1
			}
			citizen = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(os.Stderr, "unknown flag: %s\n", args[i])
				return 1
			}
			if repoPath == "" {
				repoPath = args[i]
			}
		}
		i++
	}

	if repoPath == "" {
		fmt.Fprintln(os.Stderr, "repo path required: gate check <repo-path>")
		return 1
	}

	if !pipeline.ValidLevel(level) {
		fmt.Fprintf(os.Stderr, "invalid level %q: use quick, standard, or deep\n", level)
		return 1
	}

	// Resolve citizen
	if citizen == "" {
		citizen = os.Getenv("POLIS_CITIZEN")
	}
	if citizen == "" {
		citizen = gitUserName()
	}
	if citizen == "" {
		citizen = "unknown"
	}

	v := pipeline.Run(context.Background(), repoPath, level, citizen)

	// Record verdict as a bead (if bd is available)
	if beadID := bead.Record(v); beadID != "" {
		v.Bead = beadID
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(v)
	} else {
		printPretty(v)
	}

	return v.ExitCode
}

func printUsage() {
	fmt.Println(`gate — quality gate for Polis

Usage:
  gate check <repo-path> [flags]

Flags:
  --level quick|standard|deep   Check level (default: standard)
  --json                        Output verdict as JSON
  --citizen <name>              Set actor name`)
}

func printPretty(v verdict.Verdict) {
	icon := "\033[32m✓ PASS\033[0m"
	if !v.Pass {
		icon = "\033[31m✗ FAIL\033[0m"
	}
	fmt.Printf("\n%s  %s @ %s level\n", icon, v.Repo, v.Level)
	fmt.Printf("citizen: %s\n\n", v.Citizen)

	for _, g := range v.Gates {
		gIcon := "\033[32m✓\033[0m"
		if g.Skipped {
			gIcon = "\033[33m-\033[0m"
		} else if !g.Pass {
			gIcon = "\033[31m✗\033[0m"
		}
		fmt.Printf("  %s %-20s %dms\n", gIcon, g.Name, g.DurationMs)
		if !g.Pass && !g.Skipped && g.Output != "" {
			for _, line := range strings.Split(g.Output, "\n") {
				if line != "" {
					fmt.Printf("    %s\n", line)
				}
			}
		}
	}
	if v.Bead != "" {
		fmt.Printf("\nbead: %s\n", v.Bead)
	}
	fmt.Println()
}

func gitUserName() string {
	out, err := exec.Command("git", "config", "user.name").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
