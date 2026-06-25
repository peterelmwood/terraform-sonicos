package lint

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// Lint runs every rule over the configuration and returns the findings, sorted
// for stable output (by resource address, then rule, then message).
func Lint(c *Config) []Finding {
	var findings []Finding
	for _, r := range allRules {
		findings = append(findings, r(c)...)
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Address != findings[j].Address {
			return findings[i].Address < findings[j].Address
		}
		if findings[i].Rule != findings[j].Rule {
			return findings[i].Rule < findings[j].Rule
		}
		return findings[i].Message < findings[j].Message
	})
	return findings
}

// LintJSON parses `terraform show -json` output and lints it.
func LintJSON(data []byte) ([]Finding, error) {
	cfg, err := Parse(data)
	if err != nil {
		return nil, err
	}
	return Lint(cfg), nil
}

// Counts summarizes findings by severity.
type Counts struct {
	Errors   int
	Warnings int
}

// Summarize counts findings by severity.
func Summarize(findings []Finding) Counts {
	var c Counts
	for _, f := range findings {
		switch f.Severity {
		case SeverityError:
			c.Errors++
		case SeverityWarning:
			c.Warnings++
		}
	}
	return c
}

// ReportText writes a human-readable report to w and returns the counts.
func ReportText(w io.Writer, findings []Finding) Counts {
	for _, f := range findings {
		addr := f.Address
		if addr == "" {
			addr = "(config)"
		}
		fmt.Fprintf(w, "%-7s %s  [%s]\n        %s\n", string(f.Severity), addr, f.Rule, f.Message)
	}
	counts := Summarize(findings)
	if len(findings) == 0 {
		fmt.Fprintln(w, "No issues found.")
	} else {
		fmt.Fprintf(w, "\n%d error(s), %d warning(s)\n", counts.Errors, counts.Warnings)
	}
	return counts
}

// ReportJSON writes findings as a JSON array to w and returns the counts.
func ReportJSON(w io.Writer, findings []Finding) (Counts, error) {
	if findings == nil {
		findings = []Finding{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(findings); err != nil {
		return Counts{}, err
	}
	return Summarize(findings), nil
}
