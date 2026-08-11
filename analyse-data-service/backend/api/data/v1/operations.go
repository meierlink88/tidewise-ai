package v1

const (
	OperationPublishResearchTheme            = "data.v1.publishResearchTheme"
	OperationListResearchThemes              = "data.v1.listResearchThemes"
	OperationGetResearchTheme                = "data.v1.getResearchTheme"
	OperationListResearchThemeReasoningTrees = "data.v1.listResearchThemeReasoningTrees"
	OperationGetResearchThemeReasoningTree   = "data.v1.getResearchThemeReasoningTree"
	OperationListResearchAnalysisContext     = "data.v1.listResearchAnalysisContext"
	OperationSearchResearchGraph             = "data.v1.searchResearchGraph"
	OperationGetRuntimeHealth                = "data.v1.getRuntimeHealth"
)

func BusinessOperations() []string {
	return []string{
		OperationPublishResearchTheme,
		OperationListResearchThemes,
		OperationGetResearchTheme,
		OperationListResearchThemeReasoningTrees,
		OperationGetResearchThemeReasoningTree,
		OperationListResearchAnalysisContext,
		OperationSearchResearchGraph,
		OperationGetRuntimeHealth,
	}
}
