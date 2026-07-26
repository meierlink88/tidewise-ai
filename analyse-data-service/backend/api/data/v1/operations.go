package v1

const (
	OperationPublishReviewedEvents           = "data.v1.publishReviewedEvents"
	OperationImportResearchThemes            = "data.v1.importResearchThemes"
	OperationImportResearchAnchors           = "data.v1.importResearchAnchors"
	OperationListResearchThemes              = "data.v1.listResearchThemes"
	OperationGetResearchTheme                = "data.v1.getResearchTheme"
	OperationListResearchThemeReasoningTrees = "data.v1.listResearchThemeReasoningTrees"
	OperationGetResearchThemeReasoningTree   = "data.v1.getResearchThemeReasoningTree"
	OperationListAdminRawDocuments           = "data.v1.listAdminRawDocuments"
	OperationListAdminEvents                 = "data.v1.listAdminEvents"
)

var BusinessOperations = []string{
	OperationPublishReviewedEvents,
	OperationImportResearchThemes,
	OperationImportResearchAnchors,
	OperationListResearchThemes,
	OperationGetResearchTheme,
	OperationListResearchThemeReasoningTrees,
	OperationGetResearchThemeReasoningTree,
	OperationListAdminRawDocuments,
	OperationListAdminEvents,
}
