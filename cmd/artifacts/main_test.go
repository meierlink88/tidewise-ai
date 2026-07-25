package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestArtifactCLIExposesVerifyAndRebuildCommands(t *testing.T) {
	root := t.TempDir()
	for _, command := range []string{"rebuild-index", "verify-index"} {
		t.Run(command, func(t *testing.T) {
			var output bytes.Buffer
			exitCode := run([]string{command, "--root", root}, &output)
			if exitCode != 0 {
				t.Fatalf("exit code = %d, output=%s", exitCode, output.String())
			}
			var response struct {
				Command   string `json:"command"`
				Documents int    `json:"documents"`
				Records   int    `json:"records"`
			}
			if err := json.Unmarshal(output.Bytes(), &response); err != nil {
				t.Fatalf("output = %q: %v", output.String(), err)
			}
			if response.Command != command || response.Documents != 0 || response.Records != 0 {
				t.Fatalf("response = %#v", response)
			}
		})
	}
}

func TestArtifactCLIExposesReadOnlyPollutionAudit(t *testing.T) {
	var output bytes.Buffer
	exitCode := run([]string{"audit-pollution", "--root", t.TempDir()}, &output)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, output=%s", exitCode, output.String())
	}
	var response struct {
		Command string `json:"command"`
		Report  struct {
			Documents int   `json:"documents"`
			Findings  []any `json:"findings"`
		} `json:"report"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Command != "audit-pollution" || response.Report.Documents != 0 || len(response.Report.Findings) != 0 {
		t.Fatalf("response = %#v", response)
	}
}
