package model

type ReviewStatus string

const (
	ReviewStatusCandidate ReviewStatus = "candidate"
	ReviewStatusReviewed  ReviewStatus = "reviewed"
	ReviewStatusPending   ReviewStatus = "pending"
	ReviewStatusApproved  ReviewStatus = "approved"
	ReviewStatusRejected  ReviewStatus = "rejected"
)
