package eventsemantic

import (
	"context"
	"encoding/json"
	"time"
)

type TransactionStore interface {
	InTransaction(context.Context, func(Transaction) error) error
}

type Transaction interface {
	LoadContextLeaseState(context.Context, ContextLeaseRequest, time.Time) (ContextLeaseTransactionState, error)
	SaveContextLease(context.Context, ContextLeaseWrite) error
	LoadSubmissionState(context.Context, Submission) (SubmissionTransactionState, error)
	SaveSubmission(context.Context, SubmissionWrite) error
	LoadReviewState(context.Context, ReviewSubmission) (ReviewTransactionState, error)
	SaveReview(context.Context, ReviewWrite) error
}

type ContextLeaseTransactionState struct {
	Existing             *StoredContextLease
	Event                LeaseEventState
	ActiveLeaseID        string
	ExpiredLeaseIDs      []string
	HasActiveSubmission  bool
	SupersededSubmission *SubmissionReference
}

type StoredContextLease struct {
	ContextLease
	AgentExecutionID string
	WorkerID         string
	SubmissionStatus ReviewStatus
}

type LeaseEventState struct {
	Found       bool
	EventID     string
	EventStatus string
	FactStatus  string
	InputValid  bool
}

type SubmissionReference struct {
	SubmissionID   string
	EventID        string
	ContextLeaseID string
	Status         ReviewStatus
}

type ContextLeaseWrite struct {
	Lease                  ContextLease
	AgentExecutionID       string
	WorkerID               string
	ExpireLeaseIDs         []string
	ConsumeSupersededLease bool
	Refresh                bool
	TransitionedAt         time.Time
}

type SubmissionTransactionState struct {
	Existing             *SubmissionResult
	Lease                SubmissionLeaseState
	Context              Context
	SupersededSubmission *SubmissionReference
}

type SubmissionLeaseState struct {
	Found                  bool
	EventID                string
	AgentExecutionID       string
	Status                 string
	LeaseExpiresAt         time.Time
	SupersedesSubmissionID string
}

type SubmissionWrite struct {
	SubmissionID   string
	SnapshotID     string
	CandidateIDs   SemanticCandidateIDs
	Submission     Submission
	Payload        json.RawMessage
	PayloadHash    string
	Precheck       PrecheckResult
	Status         ReviewStatus
	ConsumeLease   bool
	TransitionedAt time.Time
}

type SemanticCandidateIDs struct {
	EntityLinks     map[string]string
	VariableSignals map[string]string
	Measurements    map[string][]string
}

type ReviewTransactionState struct {
	Found            bool
	Identity         ReviewIdentity
	ExistingSnapshot *ReviewSnapshot
	Submission       *SubmissionResult
	ReviewCount      int
	RetryBudget      int
}

type ReviewIdentity struct {
	AgentExecutionID      string
	ReviewerPromptHash    string
	ReviewerModel         string
	AdjudicatorPromptHash string
	AdjudicatorModel      string
}

func (identity ReviewIdentity) Matches(submission ReviewSubmission) bool {
	switch submission.ReviewerExecutionKey {
	case identity.AgentExecutionID + ":reviewer":
		return submission.PromptHash == identity.ReviewerPromptHash &&
			submission.Model == identity.ReviewerModel
	case identity.AgentExecutionID + ":adjudicator":
		return identity.AdjudicatorPromptHash != "" &&
			submission.PromptHash == identity.AdjudicatorPromptHash &&
			submission.Model == identity.AdjudicatorModel
	default:
		return false
	}
}

type ReviewWrite struct {
	SnapshotID     string
	Submission     ReviewSubmission
	Payload        json.RawMessage
	PayloadHash    string
	Precheck       PrecheckResult
	Status         ReviewStatus
	SupersedePrior bool
	ConsumeLease   bool
	FinalizedAt    *time.Time
	TransitionedAt time.Time
}
