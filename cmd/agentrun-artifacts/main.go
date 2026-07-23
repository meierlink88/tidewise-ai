package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/artifacts"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout))
}

func run(args []string, output io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(output, "command is required")
		return 2
	}
	command := args[0]
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(output)
	root := flags.String("root", "data", "Artifact root")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(output, "unexpected arguments")
		return 2
	}
	var report any
	var err error
	switch command {
	case "verify-index":
		report, err = artifacts.VerifyIndex(*root)
	case "rebuild-index":
		report, err = artifacts.RebuildIndex(*root)
	case "audit-pollution":
		report, err = artifacts.AuditPollution(*root)
	default:
		fmt.Fprintln(output, "unknown command")
		return 2
	}
	if err != nil {
		fmt.Fprintln(output, err)
		return 1
	}
	var response any
	if indexReport, ok := report.(artifacts.IndexReport); ok {
		response = struct {
			Command string `json:"command"`
			artifacts.IndexReport
		}{Command: command, IndexReport: indexReport}
	} else {
		response = map[string]any{"command": command, "report": report}
	}
	if err := json.NewEncoder(output).Encode(response); err != nil {
		return 1
	}
	return 0
}
