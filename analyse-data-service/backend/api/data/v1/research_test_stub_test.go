package v1

import "context"

func (testDataHTTPServer) ListResearchAnalysisContext(context.Context, *ResearchAnalysisContextRequest) (*Response[ResearchAnalysisContext], error) {
	return testResponse[ResearchAnalysisContext]()
}

func (testDataHTTPServer) SearchResearchGraph(context.Context, *ResearchGraphSearchRequest) (*Response[ResearchGraphSearchResult], error) {
	return testResponse[ResearchGraphSearchResult]()
}
