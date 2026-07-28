package v1

const (
	OperationPublishReviewedEvents           = "data.v1.publishReviewedEvents"
	OperationListActiveEventTags             = "data.v1.listActiveEventTags"
	OperationImportResearchThemes            = "data.v1.importResearchThemes"
	OperationImportResearchReasoningTrees    = "data.v1.importResearchReasoningTrees"
	OperationListResearchThemes              = "data.v1.listResearchThemes"
	OperationGetResearchTheme                = "data.v1.getResearchTheme"
	OperationListResearchThemeReasoningTrees = "data.v1.listResearchThemeReasoningTrees"
	OperationGetResearchThemeReasoningTree   = "data.v1.getResearchThemeReasoningTree"
	OperationListAdminRawDocuments           = "data.v1.listAdminRawDocuments"
	OperationListAdminEvents                 = "data.v1.listAdminEvents"
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
}
