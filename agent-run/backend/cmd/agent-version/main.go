package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
	agentrunconfig "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/postgres"
)

const usage = "usage: agentrun-agent-version publish-current|withdraw-publication"

type agentVersionDocument struct {
	AgentKey string `json:"agent_key"`
	Version  string `json:"version"`
}

type publicationDocument struct {
	Added []agentVersionDocument `json:"added"`
}

func main() {
	if len(os.Args) != 2 || (os.Args[1] != "publish-current" && os.Args[1] != "withdraw-publication") {
		fail(usage)
	}
	cfg, err := agentrunconfig.LoadDatabaseOperation()
	if err != nil {
		fail("could not load AgentRun configuration")
	}
	databaseURL, err := cfg.PostgresURL()
	if err != nil {
		fail("could not build AgentRun database configuration")
	}
	ctx := context.Background()
	database, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		fail("could not open AgentRun database")
	}
	defer database.Close()
	store := postgres.New(database)
	if !store.SchemaReady(ctx) {
		fail("AgentRun database schema is not ready; run migrations first")
	}
	publisher, err := agentrun.NewAgentVersionPublisher(store)
	if err != nil {
		fail("could not construct Agent Version publisher")
	}
	if os.Args[1] == "withdraw-publication" {
		publication, err := decodePublication(os.Stdin)
		if err != nil {
			fail("Agent Version publication record is invalid")
		}
		if err := publisher.Withdraw(ctx, publication); err != nil {
			fail("could not withdraw candidate Agent Versions")
		}
		fmt.Println("AgentRun candidate Agent Versions are withdrawn")
		return
	}
	publication, err := publisher.PublishCurrent(ctx, currentAgentVersions())
	if err != nil {
		fail("could not publish current Agent Versions")
	}
	if err := json.NewEncoder(os.Stdout).Encode(encodePublication(publication)); err != nil {
		fail("could not encode Agent Version publication record")
	}
}

func currentAgentVersions() []agentrun.AgentVersion {
	return []agentrun.AgentVersion{
		{AgentKey: collector.AgentKey, Version: collector.AgentVersion},
		{AgentKey: eventfact.AgentKey, Version: eventfact.AgentVersion},
		{AgentKey: eventsemantic.AgentKey, Version: eventsemantic.AgentVersion},
	}
}

func encodePublication(publication agentrun.AgentVersionPublication) publicationDocument {
	document := publicationDocument{Added: make([]agentVersionDocument, 0, len(publication.Added))}
	for _, version := range publication.Added {
		document.Added = append(document.Added, agentVersionDocument{
			AgentKey: version.AgentKey,
			Version:  version.Version,
		})
	}
	return document
}

func decodePublication(reader io.Reader) (agentrun.AgentVersionPublication, error) {
	payload, err := io.ReadAll(io.LimitReader(reader, 64*1024+1))
	if err != nil || len(payload) > 64*1024 {
		return agentrun.AgentVersionPublication{}, fmt.Errorf("read publication record")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var document publicationDocument
	if err := decoder.Decode(&document); err != nil {
		return agentrun.AgentVersionPublication{}, fmt.Errorf("decode publication record: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return agentrun.AgentVersionPublication{}, fmt.Errorf("publication record has trailing content")
	}
	publication := agentrun.AgentVersionPublication{
		Added: make([]agentrun.AgentVersion, 0, len(document.Added)),
	}
	for _, version := range document.Added {
		publication.Added = append(publication.Added, agentrun.AgentVersion{
			AgentKey: version.AgentKey,
			Version:  version.Version,
		})
	}
	return publication, nil
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
