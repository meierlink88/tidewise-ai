package v1

const (
	OperationPublishReviewedEvents           = "data.v1.publishReviewedEvents"
	OperationListActiveEventTags             = "data.v1.listActiveEventTags"
	OperationPublishResearchTheme            = "data.v1.publishResearchTheme"
	OperationListResearchThemes              = "data.v1.listResearchThemes"
	OperationGetResearchTheme                = "data.v1.getResearchTheme"
	OperationListResearchThemeReasoningTrees = "data.v1.listResearchThemeReasoningTrees"
	OperationGetResearchThemeReasoningTree   = "data.v1.getResearchThemeReasoningTree"
	OperationListAdminRawDocuments           = "data.v1.listAdminRawDocuments"
	OperationListAdminEvents                 = "data.v1.listAdminEvents"
	OperationListEligibleEventSemanticEvents = "data.v1.listEligibleEventSemanticEvents"
	OperationCreateEventSemanticContextLease = "data.v1.createEventSemanticContextLease"
	OperationGetEventSemanticContext         = "data.v1.getEventSemanticContext"
	OperationCreateEventSemanticSubmission   = "data.v1.createEventSemanticSubmission"
	OperationSubmitEventSemanticReview       = "data.v1.submitEventSemanticReview"
	OperationGetEventSemantics               = "data.v1.getEventSemantics"
	OperationListResearchAnalysisContext     = "data.v1.listResearchAnalysisContext"
	OperationSearchResearchGraph             = "data.v1.searchResearchGraph"
	OperationGetRuntimeHealth                = "data.v1.getRuntimeHealth"
)

var BusinessOperations = []string{
	OperationPublishReviewedEvents,
	OperationListActiveEventTags,
	OperationPublishResearchTheme,
	OperationListResearchThemes,
	OperationGetResearchTheme,
	OperationListResearchThemeReasoningTrees,
	OperationGetResearchThemeReasoningTree,
	OperationListAdminRawDocuments,
	OperationListAdminEvents,
	OperationListEligibleEventSemanticEvents,
	OperationCreateEventSemanticContextLease,
	OperationGetEventSemanticContext,
	OperationCreateEventSemanticSubmission,
	OperationSubmitEventSemanticReview,
	OperationGetEventSemantics,
	OperationListResearchAnalysisContext,
	OperationSearchResearchGraph,
	OperationGetRuntimeHealth,
}
