package main

import (
	"bytes"
	"strings"
	"testing"
)

const cleanInput = `{"planned_values":{"root_module":{"resources":[
  {"address":"sonicos_zone.dmz","type":"sonicos_zone","name":"dmz","values":{"name":"DMZ","security_type":"public"}}
]}}}`

const errorInput = `{"planned_values":{"root_module":{"resources":[
  {"address":"sonicos_access_rule.r","type":"sonicos_access_rule","name":"r","values":{"name":"r","from":"LAN","to":"WAN","action":"allow","enable":true,"source_name":"missing"}}
]}}}`

func TestRunLintCleanExitsZero(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"lint"}, strings.NewReader(cleanInput), &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d (stderr: %s)", code, errOut.String())
	}
	if !strings.Contains(out.String(), "No issues found") {
		t.Errorf("expected clean message, got %q", out.String())
	}
}

func TestRunLintErrorExitsOne(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"lint"}, strings.NewReader(errorInput), &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(out.String(), "referential.missing_address_object") {
		t.Errorf("expected referential finding in output, got %q", out.String())
	}
}

func TestRunLintJSONFormat(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"lint", "-format", "json"}, strings.NewReader(errorInput), &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.HasPrefix(strings.TrimSpace(out.String()), "[") {
		t.Errorf("expected JSON array output, got %q", out.String())
	}
}

func TestRunLintStrictFailsOnWarnings(t *testing.T) {
	// An any→any allow is a warning, not an error.
	const warnInput = `{"planned_values":{"root_module":{"resources":[
      {"address":"sonicos_access_rule.r","type":"sonicos_access_rule","name":"r","values":{"name":"r","from":"LAN","to":"WAN","action":"allow","enable":true}}
    ]}}}`

	var out, errOut bytes.Buffer
	if code := run([]string{"lint"}, strings.NewReader(warnInput), &out, &errOut); code != 0 {
		t.Fatalf("warnings alone should exit 0, got %d", code)
	}
	out.Reset()
	if code := run([]string{"lint", "-strict"}, strings.NewReader(warnInput), &out, &errOut); code != 1 {
		t.Fatalf("strict should exit 1 on warnings, got %d", code)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"bogus"}, strings.NewReader(""), &out, &errOut); code != 2 {
		t.Fatalf("expected exit 2 for unknown command, got %d", code)
	}
}
