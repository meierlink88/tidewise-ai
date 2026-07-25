package artifacts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/agents/collector"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/agents/collector/materialization"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/biz/platform"
)

const publicationPlanSchema = "collector_artifact_publication.v1"

type PublicationRepository interface {
	PreparePublication(context.Context, agentrun.PublicationReference) error
	ListPreparedPublications(context.Context) ([]agentrun.PublicationReference, error)
	CommitPreparedPublication(context.Context, agentrun.PublicationReference, agentrun.ExecutionCompletion) error
}

type publicationItem struct {
	Kind   string `json:"kind"`
	Source string `json:"source"`
	Target string `json:"target"`
	SHA256 string `json:"sha256"`
}

type publicationPlan struct {
	Schema              string                   `json:"schema"`
	ExecutionID         string                   `json:"execution_id"`
	TerminalStatus      agentrun.ExecutionStatus `json:"terminal_status"`
	StopReason          string                   `json:"stop_reason"`
	ErrorCode           string                   `json:"error_code,omitempty"`
	ErrorSummary        string                   `json:"error_summary,omitempty"`
	CandidateCounts     map[string]int           `json:"candidate_counts"`
	Artifacts           map[string]string        `json:"artifacts"`
	CompletedAt         string                   `json:"completed_at"`
	PreviousIndexSHA256 string                   `json:"previous_index_sha256,omitempty"`
	Items               []publicationItem        `json:"items"`
}

func preparePublicationPlan(
	root string,
	result collector.Result,
	staged collector.Result,
	stagedDocuments map[string]string,
	previousIndexSHA256 string,
	status agentrun.ExecutionStatus,
	completedAt time.Time,
) (agentrun.PublicationReference, publicationPlan, error) {
	plan := publicationPlan{
		Schema: publicationPlanSchema, ExecutionID: result.RunID,
		TerminalStatus: status, StopReason: result.StopReason,
		CandidateCounts: materialization.CandidateCounts(result.Stats),
		Artifacts: map[string]string{
			"documents": result.Documents, "index": result.Index,
			"candidates": result.Candidates, "summary": result.Summary, "manifest": result.Manifest,
		},
		CompletedAt:         completedAt.UTC().Format(time.RFC3339Nano),
		PreviousIndexSHA256: previousIndexSHA256,
	}
	if status == agentrun.StatusFailed {
		plan.ErrorCode = "all_connectors_failed"
		plan.ErrorSummary = "All Connector invocations failed"
	}
	documentTargets := make([]string, 0, len(stagedDocuments))
	for target := range stagedDocuments {
		documentTargets = append(documentTargets, target)
	}
	sort.Strings(documentTargets)
	for _, target := range documentTargets {
		item, err := buildPublicationItem("document", stagedDocuments[target], target)
		if err != nil {
			return agentrun.PublicationReference{}, publicationPlan{}, err
		}
		plan.Items = append(plan.Items, item)
	}
	for _, item := range []struct {
		kind, source, target string
	}{
		{"candidates", staged.Candidates, result.Candidates},
		{"summary", staged.Summary, result.Summary},
		{"index", staged.Index, result.Index},
		{"manifest", staged.Manifest, result.Manifest},
	} {
		built, err := buildPublicationItem(item.kind, item.source, item.target)
		if err != nil {
			return agentrun.PublicationReference{}, publicationPlan{}, err
		}
		plan.Items = append(plan.Items, built)
	}
	encoded, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return agentrun.PublicationReference{}, publicationPlan{}, fmt.Errorf("encode Artifact publication plan: %w", err)
	}
	encoded = append(encoded, '\n')
	planPath := filepath.Join(root, ".pending", result.RunID, "plan.json")
	if err := atomicWrite(planPath, encoded, 0o600); err != nil {
		return agentrun.PublicationReference{}, publicationPlan{}, fmt.Errorf("write Artifact publication plan: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return agentrun.PublicationReference{
		ExecutionID: result.RunID, PlanPath: planPath,
		PlanSHA256: hex.EncodeToString(sum[:]), PreparedAt: completedAt.UTC(),
	}, plan, nil
}

func buildPublicationItem(kind, source, target string) (publicationItem, error) {
	digest, err := fileSHA256(source)
	if err != nil {
		return publicationItem{}, fmt.Errorf("hash staged %s Artifact: %w", kind, err)
	}
	return publicationItem{Kind: kind, Source: source, Target: target, SHA256: digest}, nil
}

func publishPreparedPlan(ctx context.Context, root string, plan publicationPlan, beforeStep func(string) error) error {
	if err := validatePublicationPlan(root, plan); err != nil {
		return err
	}
	for position, item := range plan.Items {
		if err := ctx.Err(); err != nil {
			return err
		}
		if beforeStep != nil {
			if err := beforeStep(item.Kind); err != nil {
				return err
			}
		}
		if item.Kind == "index" {
			if err := publishIndexItem(item, plan.PreviousIndexSHA256); err != nil {
				return err
			}
		} else if err := publishImmutableItem(item); err != nil {
			return err
		}
		if item.Kind == "manifest" && position != len(plan.Items)-1 {
			return fmt.Errorf("Artifact publication manifest is not last")
		}
	}
	return nil
}

func publishImmutableItem(item publicationItem) error {
	sourceHash, err := fileSHA256(item.Source)
	if err != nil {
		return fmt.Errorf("read staged %s Artifact: %w", item.Kind, err)
	}
	if sourceHash != item.SHA256 {
		return fmt.Errorf("staged %s Artifact hash mismatch", item.Kind)
	}
	if current, err := fileSHA256(item.Target); err == nil {
		if current == item.SHA256 {
			return nil
		}
		return fmt.Errorf("published %s Artifact conflicts with plan", item.Kind)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect published %s Artifact: %w", item.Kind, err)
	}
	return atomicCopy(item.Source, item.Target, 0o644)
}

func publishIndexItem(item publicationItem, previousHash string) error {
	sourceHash, err := fileSHA256(item.Source)
	if err != nil {
		return fmt.Errorf("read staged index Artifact: %w", err)
	}
	if sourceHash != item.SHA256 {
		return fmt.Errorf("staged index Artifact hash mismatch")
	}
	currentHash, err := fileSHA256(item.Target)
	if err == nil {
		if currentHash == item.SHA256 {
			return nil
		}
		if currentHash != previousHash {
			return fmt.Errorf("published index Artifact conflicts with plan")
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect published index Artifact: %w", err)
	} else if previousHash != "" {
		return fmt.Errorf("published index Artifact is missing")
	}
	return atomicCopy(item.Source, item.Target, 0o644)
}

func loadPublicationPlan(root string, reference agentrun.PublicationReference) (publicationPlan, error) {
	if !pathWithin(root, reference.PlanPath) {
		return publicationPlan{}, fmt.Errorf("Artifact publication plan escapes root")
	}
	payload, err := os.ReadFile(reference.PlanPath)
	if err != nil {
		return publicationPlan{}, fmt.Errorf("read Artifact publication plan: %w", err)
	}
	sum := sha256.Sum256(payload)
	if hex.EncodeToString(sum[:]) != reference.PlanSHA256 {
		return publicationPlan{}, fmt.Errorf("Artifact publication plan hash mismatch")
	}
	var plan publicationPlan
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&plan); err != nil {
		return publicationPlan{}, fmt.Errorf("decode Artifact publication plan: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return publicationPlan{}, fmt.Errorf("decode Artifact publication plan: trailing content")
	}
	if plan.ExecutionID != reference.ExecutionID {
		return publicationPlan{}, fmt.Errorf("Artifact publication plan Execution mismatch")
	}
	if err := validatePublicationPlan(root, plan); err != nil {
		return publicationPlan{}, err
	}
	return plan, nil
}

func validatePublicationPlan(root string, plan publicationPlan) error {
	if plan.Schema != publicationPlanSchema || plan.ExecutionID == "" ||
		plan.StopReason == "" || len(plan.Items) < 4 {
		return fmt.Errorf("invalid Artifact publication plan")
	}
	switch plan.TerminalStatus {
	case agentrun.StatusSucceeded, agentrun.StatusSucceededNoChange, agentrun.StatusPartiallySucceeded:
	case agentrun.StatusFailed:
		if plan.ErrorCode == "" || plan.ErrorSummary == "" {
			return fmt.Errorf("failed Artifact publication is missing a safe error")
		}
	default:
		return fmt.Errorf("invalid Artifact publication terminal status")
	}
	if plan.CandidateCounts["results_pending"] != 0 {
		return fmt.Errorf("Artifact publication has pending Candidates")
	}
	if plan.Items[len(plan.Items)-1].Kind != "manifest" {
		return fmt.Errorf("Artifact publication manifest is not last")
	}
	for _, item := range plan.Items {
		if item.Source == "" || item.Target == "" || !validHex(item.SHA256, 64) ||
			!pathWithin(root, item.Source) || !pathWithin(root, item.Target) {
			return fmt.Errorf("invalid Artifact publication item")
		}
	}
	return nil
}

func ReconcilePreparedPublications(ctx context.Context, root string, repository PublicationRepository) error {
	references, err := repository.ListPreparedPublications(ctx)
	if err != nil {
		return err
	}
	for _, reference := range references {
		plan, err := loadPublicationPlan(root, reference)
		if err != nil {
			return err
		}
		if err := publishPreparedPlan(ctx, root, plan, nil); err != nil {
			return err
		}
		completion, err := planCompletion(plan)
		if err != nil {
			return err
		}
		if err := retryPublicationCall(ctx, func() error {
			return repository.CommitPreparedPublication(ctx, reference, completion)
		}); err != nil {
			return err
		}
		_ = os.RemoveAll(filepath.Dir(reference.PlanPath))
	}
	return removeOrphanedPendingDirectories(root)
}

func removeOrphanedPendingDirectories(root string) error {
	pendingRoot := filepath.Join(root, ".pending")
	entries, err := os.ReadDir(pendingRoot)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("scan orphaned pending Artifact directories: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if err := os.RemoveAll(filepath.Join(pendingRoot, entry.Name())); err != nil {
			return fmt.Errorf("remove orphaned pending Artifact directory: %w", err)
		}
	}
	return nil
}

func retryPublicationCall(ctx context.Context, call func() error) error {
	var last error
	for range 3 {
		if err := ctx.Err(); err != nil {
			return err
		}
		last = call()
		if last == nil {
			return nil
		}
	}
	return last
}

func planCompletion(plan publicationPlan) (agentrun.ExecutionCompletion, error) {
	completedAt, err := time.Parse(time.RFC3339Nano, plan.CompletedAt)
	if err != nil {
		return agentrun.ExecutionCompletion{}, fmt.Errorf("invalid Artifact publication completion time")
	}
	return agentrun.ExecutionCompletion{
		ExecutionID: plan.ExecutionID, Status: plan.TerminalStatus,
		StopReason: plan.StopReason, ErrorCode: plan.ErrorCode, ErrorSummary: plan.ErrorSummary,
		CandidateCounts: plan.CandidateCounts,
		Artifacts:       plan.Artifacts, CompletedAt: completedAt,
	}, nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func atomicCopy(source, target string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".tmp-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := io.Copy(temporary, input); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, target)
}

func pathWithin(root, path string) bool {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(absoluteRoot, absolutePath)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
