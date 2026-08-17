package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	data "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data"
)

var categories = []string{
	"correct_reject", "abox_missing", "mention_extraction_miss",
	"retrieval_miss", "selector_false_reject", "review_reject", "model_contract_failure",
}

var classificationPrecedence = []string{
	"model_contract_failure", "mention_extraction_miss", "retrieval_miss",
	"selector_false_reject", "review_reject", "abox_missing", "correct_reject",
}

type acceptanceReport struct {
	Runs []acceptanceRun `json:"runs"`
}

type referenceReport struct {
	Events []struct {
		EventID  string `json:"event_id"`
		Mentions []struct {
			Text string `json:"text"`
		} `json:"mentions"`
	} `json:"events"`
}

type acceptanceRun struct {
	EventID, Title, Status, ErrorCode string
	FixedEvent                        bool
	EntityRejected                    int
	EntityDecisions                   []candidateDecision
	StageAudit                        stageAudit
}

type candidateDecision struct {
	CandidateKey string `json:"candidate_key"`
	Status       string `json:"status"`
}

type stageAudit struct {
	Mentions      []mentionAudit      `json:"mentions"`
	CandidateSets []candidateSetAudit `json:"candidate_sets"`
	Selections    []selectionAudit    `json:"selections"`
}

type mentionAudit struct {
	CandidateKey string `json:"candidate_key"`
	Mention      string `json:"mention"`
}

type candidateSetAudit struct {
	CandidateKey string           `json:"candidate_key"`
	Candidates   []candidateAudit `json:"candidates"`
}

type candidateAudit struct {
	EntityID string `json:"entity_id"`
}

type selectionAudit struct {
	CandidateKey string `json:"candidate_key"`
	EntityID     string `json:"entity_id"`
	ReasonCode   string `json:"reason_code"`
	NoMatch      bool   `json:"no_match"`
}

type entityIdentity struct {
	ID, EntityType, Name, CanonicalName, Status string
	Aliases                                     []string
}

type eventText struct {
	Title, Corpus string
}

type classification struct {
	EventID  string `json:"event_id"`
	Title    string `json:"title"`
	Category string `json:"category"`
	Detail   string `json:"detail"`
}

type auditOutput struct {
	ContractVersion     string           `json:"contract_version"`
	GeneratedAt         string           `json:"generated_at"`
	InputSHA256         string           `json:"input_sha256"`
	ReferenceSHA256     string           `json:"reference_sha256"`
	ClassificationOrder []string         `json:"classification_order"`
	Counts              map[string]int   `json:"counts"`
	Events              []classification `json:"events"`
}

func main() {
	inputPath, referencePath, outputPath, err := parseOptions(os.Args[1:])
	if err != nil {
		fail(err)
	}
	input, err := os.ReadFile(inputPath)
	if err != nil {
		fail(errors.New("could not read acceptance report"))
	}
	var report acceptanceReport
	if err := json.Unmarshal(input, &report); err != nil {
		fail(errors.New("acceptance report is invalid"))
	}
	referenceInput, err := os.ReadFile(referencePath)
	if err != nil {
		fail(errors.New("could not read frozen reference mentions"))
	}
	var reference referenceReport
	if err := json.Unmarshal(referenceInput, &reference); err != nil {
		fail(errors.New("frozen reference mentions are invalid"))
	}
	referenceByEvent := make(map[string][]string, len(reference.Events))
	for _, event := range reference.Events {
		for _, mention := range event.Mentions {
			referenceByEvent[event.EventID] = append(referenceByEvent[event.EventID], mention.Text)
		}
	}
	config, err := conf.LoadDatabaseOperation()
	if err != nil {
		fail(errors.New("could not load Data database configuration"))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	database, err := data.OpenPostgres(ctx, config)
	if err != nil {
		fail(errors.New("could not open Data database"))
	}
	defer database.Close()
	entities, err := loadEntities(ctx, database)
	if err != nil {
		fail(errors.New("could not read formal Entity identities"))
	}
	result := auditOutput{
		ContractVersion: "event-semantic-v3-reject-audit.v1", GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		InputSHA256: sha256Hex(input), ReferenceSHA256: sha256Hex(referenceInput),
		ClassificationOrder: append([]string(nil), classificationPrecedence...), Counts: make(map[string]int, len(categories)), Events: []classification{},
	}
	for _, category := range categories {
		result.Counts[category] = 0
	}
	for index, run := range report.Runs {
		if run.FixedEvent || index >= 100 || (run.Status != "rejected" && run.Status != "failed") {
			continue
		}
		text, loadErr := loadEventText(ctx, database, run.EventID)
		if loadErr != nil {
			fail(errors.New("could not read Event text for reject audit"))
		}
		category, detail := classify(run, text, referenceByEvent[run.EventID], entities)
		result.Counts[category]++
		result.Events = append(result.Events, classification{run.EventID, text.Title, category, detail})
	}
	sort.Slice(result.Events, func(i, j int) bool { return result.Events[i].EventID < result.Events[j].EventID })
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fail(errors.New("could not encode reject audit"))
	}
	if err := os.WriteFile(outputPath, append(encoded, '\n'), 0o600); err != nil {
		fail(errors.New("could not write reject audit"))
	}
}

func parseOptions(arguments []string) (string, string, string, error) {
	flags := flag.NewFlagSet("event-semantic-acceptance-audit", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	input := flags.String("input", "", "Event Semantic acceptance JSON")
	reference := flags.String("reference-mentions", "", "frozen system-external mention reference JSON")
	output := flags.String("output", "", "new reject audit JSON")
	if err := flags.Parse(arguments); err != nil {
		return "", "", "", err
	}
	if *input == "" || *reference == "" || *output == "" || flags.NArg() != 0 {
		return "", "", "", errors.New("input, reference-mentions and output are required")
	}
	return *input, *reference, *output, nil
}

func loadEntities(ctx context.Context, database *sql.DB) ([]entityIdentity, error) {
	rows, err := database.QueryContext(ctx, `
		SELECT identity.id, identity.object_type, identity.name, identity.canonical_name,
		       identity.aliases_json, identity.status
		FROM (
			SELECT entity.id::text id, entity.entity_type::text object_type,
			       entity.name, entity.canonical_name,
			       array_to_json(entity.aliases)::text aliases_json, entity.status::text status
			FROM entity_nodes entity
			UNION ALL
			SELECT industry.id, 'industry', industry.name, industry.name,
			       array_to_json(industry.aliases)::text, 'active'
			FROM industry
			UNION ALL
			SELECT concept.id, 'concept', concept.name, concept.name,
			       array_to_json(concept.aliases)::text, 'active'
			FROM concept
			UNION ALL
			SELECT country.id, 'country', country.name, country.name,
			       array_to_json(ARRAY[country.name_en])::text, 'active'
			FROM countries country
		) identity
		ORDER BY identity.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []entityIdentity{}
	for rows.Next() {
		var item entityIdentity
		var aliasesJSON string
		if err := rows.Scan(&item.ID, &item.EntityType, &item.Name, &item.CanonicalName, &aliasesJSON, &item.Status); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(aliasesJSON), &item.Aliases); err != nil || item.Aliases == nil {
			return nil, errors.New("formal Entity aliases are invalid")
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func loadEventText(ctx context.Context, database *sql.DB, eventID string) (eventText, error) {
	var result eventText
	err := database.QueryRowContext(ctx, `
		SELECT event.title,
		       concat_ws(' ', event.title, event.summary,
		           COALESCE(string_agg(concat_ws(' ', document.title, evidence.evidence_statement), ' '), ''))
		FROM events event
		LEFT JOIN event_sources evidence ON evidence.event_id = event.id
		LEFT JOIN raw_documents document ON document.id = evidence.raw_document_id
		WHERE event.id = $1
		GROUP BY event.id, event.title, event.summary`, eventID).Scan(&result.Title, &result.Corpus)
	return result, err
}

func classify(run acceptanceRun, text eventText, referenceMentions []string, entities []entityIdentity) (string, string) {
	if run.ErrorCode == "event_semantic_model_contract_invalid" {
		return "model_contract_failure", run.ErrorCode
	}
	mentions := make(map[string]string, len(run.StageAudit.Mentions))
	for _, mention := range run.StageAudit.Mentions {
		mentions[mention.CandidateKey] = mention.Mention
	}
	formalInText := expectedFormalIdentities(text.Corpus, referenceMentions, entities)
	for _, entity := range formalInText {
		if entity.Status == "active" && !identityExtracted(entity, mentions) {
			return "mention_extraction_miss", "formal in-scope identity was not extracted: " + entity.CanonicalName
		}
	}
	selections := make(map[string]selectionAudit, len(run.StageAudit.Selections))
	for _, selection := range run.StageAudit.Selections {
		selections[selection.CandidateKey] = selection
	}
	for key, mention := range mentions {
		for _, entity := range exactIdentities(mention, entities) {
			if entity.Status != "active" {
				continue
			}
			if !candidateSetHasID(run.StageAudit.CandidateSets, key, entity.ID) {
				return "retrieval_miss", "formal extracted identity was absent from Qdrant candidates: " + entity.CanonicalName
			}
			if selection := selections[key]; selection.NoMatch || selection.EntityID != entity.ID {
				return "selector_false_reject", "formal Qdrant identity was not selected: " + entity.CanonicalName
			}
		}
	}
	if run.EntityRejected > 0 {
		return "review_reject", fmt.Sprintf("%d entity candidates rejected by Data/AI review", run.EntityRejected)
	}
	for _, selection := range selections {
		if selection.ReasonCode == "identity_projection_gap" {
			return "abox_missing", "entity-like extracted mentions have no formal canonical/alias identity"
		}
	}
	return "correct_reject", "no acceptable formal identity after deterministic candidate audit"
}

func expectedFormalIdentities(text string, referenceMentions []string, entities []entityIdentity) []entityIdentity {
	haystack := normalize(text)
	result := []entityIdentity{}
	seen := make(map[string]struct{})
	hasReferenceCountry := false
	for _, mention := range referenceMentions {
		for _, entity := range exactIdentities(mention, entities) {
			if entity.EntityType == "country" {
				hasReferenceCountry = true
			}
			if _, exists := seen[entity.ID]; !exists {
				seen[entity.ID] = struct{}{}
				result = append(result, entity)
			}
		}
	}
	if !hasReferenceCountry {
		return result
	}
	for _, entity := range entities {
		if entity.EntityType != "country" {
			continue
		}
		// The supplemental pass deliberately excludes aliases. Short Latin aliases
		// such as "GA" otherwise match arbitrary words in the Event corpus.
		for _, name := range []string{entity.Name, entity.CanonicalName} {
			needle := normalize(name)
			if len([]rune(needle)) >= 2 && strings.Contains(haystack, needle) {
				if _, exists := seen[entity.ID]; !exists {
					seen[entity.ID] = struct{}{}
					result = append(result, entity)
				}
				break
			}
		}
	}
	return result
}

func identityExtracted(entity entityIdentity, mentions map[string]string) bool {
	for _, mention := range mentions {
		for _, name := range entityNames(entity) {
			if normalize(mention) == normalize(name) {
				return true
			}
		}
	}
	return false
}

func exactIdentities(mention string, entities []entityIdentity) []entityIdentity {
	want := normalize(mention)
	result := []entityIdentity{}
	for _, entity := range entities {
		for _, name := range entityNames(entity) {
			if normalize(name) == want {
				result = append(result, entity)
				break
			}
		}
	}
	return result
}

func entityNames(entity entityIdentity) []string {
	return append([]string{entity.Name, entity.CanonicalName}, entity.Aliases...)
}

func candidateSetHasID(sets []candidateSetAudit, key, entityID string) bool {
	for _, set := range sets {
		if set.CandidateKey != key {
			continue
		}
		for _, candidate := range set.Candidates {
			if candidate.EntityID == entityID {
				return true
			}
		}
	}
	return false
}

func normalize(value string) string {
	var builder strings.Builder
	for _, character := range strings.ToLower(strings.TrimSpace(value)) {
		if !unicode.IsSpace(character) && !unicode.IsPunct(character) {
			builder.WriteRune(character)
		}
	}
	return builder.String()
}

func sha256Hex(value []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(value))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
