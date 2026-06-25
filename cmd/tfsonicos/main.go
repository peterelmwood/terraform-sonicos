// Command tfsonicos provides auxiliary tooling for the SonicOS Terraform
// provider. Its first subcommand, "lint", statically analyzes a SonicOS
// configuration for firewall rules it must or should obey.
//
// Usage:
//
//	terraform show -json plan.tfplan | tfsonicos lint
//	terraform show -json            | tfsonicos lint        # current state
//	tfsonicos lint plan.json
//	tfsonicos lint -format json plan.json
//
// The linter reports referential errors (rules referencing undefined objects),
// duplicate names, shadowed/unreachable rules, zone inconsistencies, and
// overly-permissive rules. It exits non-zero when any error-severity finding is
// present (use -strict to also fail on warnings), so it can gate CI.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/peterelmwood/terraform-sonicos/internal/lint"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	switch args[0] {
	case "lint":
		return runLint(args[1:], stdin, stdout, stderr)
	case "-h", "--help", "help":
		fmt.Fprintln(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n%s\n", args[0], usage)
		return 2
	}
}

const usage = `tfsonicos - tooling for the SonicOS Terraform provider

Usage:
  tfsonicos lint [-format text|json] [-strict] [FILE]

  Lints a SonicOS configuration read from FILE, or from stdin when FILE is
  omitted. Input is the JSON produced by 'terraform show -json' of a plan file
  or of current state.

Flags:
  -format   Output format: text (default) or json.
  -strict   Exit non-zero on warnings as well as errors.

Examples:
  terraform show -json plan.tfplan | tfsonicos lint
  tfsonicos lint -format json plan.json`

func runLint(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	format := "text"
	strict := false
	var file string

	// Minimal flag parsing so flags may appear before or after the file arg.
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-format", "--format":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "-format requires a value (text|json)")
				return 2
			}
			i++
			format = args[i]
		case "-strict", "--strict":
			strict = true
		case "-h", "--help":
			fmt.Fprintln(stdout, usage)
			return 0
		default:
			if len(args[i]) > 0 && args[i][0] == '-' {
				fmt.Fprintf(stderr, "unknown flag %q\n", args[i])
				return 2
			}
			file = args[i]
		}
	}
	if format != "text" && format != "json" {
		fmt.Fprintf(stderr, "invalid -format %q (want text or json)\n", format)
		return 2
	}

	var (
		data []byte
		err  error
	)
	if file == "" {
		data, err = io.ReadAll(stdin)
	} else {
		data, err = os.ReadFile(file)
	}
	if err != nil {
		fmt.Fprintf(stderr, "reading input: %v\n", err)
		return 2
	}

	findings, err := lint.LintJSON(data)
	if err != nil {
		fmt.Fprintf(stderr, "lint: %v\n", err)
		return 2
	}

	var counts lint.Counts
	if format == "json" {
		counts, err = lint.ReportJSON(stdout, findings)
		if err != nil {
			fmt.Fprintf(stderr, "writing report: %v\n", err)
			return 2
		}
	} else {
		counts = lint.ReportText(stdout, findings)
	}

	if counts.Errors > 0 || (strict && counts.Warnings > 0) {
		return 1
	}
	return 0
}
