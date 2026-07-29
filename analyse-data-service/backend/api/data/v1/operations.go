package v1

const (
	OperationPublishReviewedEvents            = "data.v1.publishReviewedEvents"
	OperationListActiveEventTags              = "data.v1.listActiveEventTags"
	OperationImportResearchThemes             = "data.v1.importResearchThemes"
	OperationImportResearchReasoningTrees     = "data.v1.importResearchReasoningTrees"
	OperationListResearchThemes               = "data.v1.listResearchThemes"
	OperationGetResearchTheme                 = "data.v1.getResearchTheme"
	OperationListResearchThemeReasoningTrees  = "data.v1.listResearchThemeReasoningTrees"
	OperationGetResearchThemeReasoningTree    = "data.v1.getResearchThemeReasoningTree"
	OperationListAdminRawDocuments            = "data.v1.listAdminRawDocuments"
	OperationListAdminEvents                  = "data.v1.listAdminEvents"
	OperationListEligibleEventSemanticEvents  = "data.v1.listEligibleEventSemanticEvents"
	OperationCreateEventSemanticContextLease  = "data.v1.createEventSemanticContextLease"
	OperationGetEventSemanticContext          = "data.v1.getEventSemanticContext"
	OperationResolveEventSemanticEntities     = "data.v1.resolveEventSemanticEntities"
	OperationSearchEventSemanticDirectTargets = "data.v1.searchEventSemanticDirectTargets"
	OperationCreateEventSemanticSubmission    = "data.v1.createEventSemanticSubmission"
	OperationSubmitEventSemanticReview        = "data.v1.submitEventSemanticReview"
	OperationGetEventSemantics                = "data.v1.getEventSemantics"
)

var BusinessOperations = []string{
	OperationPublishReviewedEvents,
	OperationListActiveEventTags,
	OperationImportResearchThemes,
	OperationImportResearchReasoningTrees,
	OperationListResearchThemes,
	OperationGetResearchTheme,
	OperationListResearchThemeReasoningTrees,
	OperationGetResearchThemeReasoningTree,
	OperationListAdminRawDocuments,
	OperationListAdminEvents,
	OperationListEligibleEventSemanticEvents,
	OperationCreateEventSemanticContextLease,
	OperationGetEventSemanticContext,
	OperationResolveEventSemanticEntities,
	OperationSearchEventSemanticDirectTargets,
	OperationCreateEventSemanticSubmission,
	OperationSubmitEventSemanticReview,
	OperationGetEventSemantics,
}
