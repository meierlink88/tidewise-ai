package eventsemantic

import "context"

// TransactionStore owns the atomic persistence seams for lease, submission and review state changes.
type TransactionStore interface {
	CreateContextLease(context.Context, ContextLeaseRequest) (ContextLease, error)
	CreateSubmission(context.Context, Submission, PrecheckResult, []byte, string) (SubmissionResult, error)
	SubmitReview(context.Context, ReviewSubmission, []byte, string) (SubmissionResult, error)
}
