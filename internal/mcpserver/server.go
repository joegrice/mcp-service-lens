package mcpserver

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"mcp-service-lens/internal/docs"
	"mcp-service-lens/internal/search"
	"mcp-service-lens/internal/services"
)

type Server struct {
	registry *services.Registry
	reader   *docs.Reader
}

type EmptyInput struct{}

type ServiceInput struct {
	Service string `json:"service" jsonschema:"configured repository name"`
}

type TraceInput struct {
	Query      string  `json:"query" jsonschema:"correlation ID, keyword, or endpoint path"`
	Service    *string `json:"service,omitempty" jsonschema:"optional configured service name; searches all services when omitted"`
	MaxResults *int    `json:"max_results,omitempty" jsonschema:"optional maximum number of matching lines to return; must be between 1 and 1000"`
}

type DocumentationSearchInput struct {
	Query      string  `json:"query" jsonschema:"text to search for in documentation files"`
	Service    *string `json:"service,omitempty" jsonschema:"optional configured service name; searches all services when omitted"`
	MaxResults *int    `json:"max_results,omitempty" jsonschema:"optional maximum number of matching lines to return; must be between 1 and 1000"`
}

type InvestigationInput struct {
	Query      string  `json:"query" jsonschema:"natural-language question, concept, or code symbol to investigate"`
	Service    *string `json:"service,omitempty" jsonschema:"optional configured service name; searches all services when omitted"`
	MaxResults *int    `json:"max_results,omitempty" jsonschema:"optional maximum number of evidence lines to return; defaults to 25"`
}

type ServiceOutput struct {
	Services []ServiceSummary `json:"services"`
}

type ServiceSummary struct {
	Name           string   `json:"name"`
	RootPath       string   `json:"root_path"`
	LogDirectories []string `json:"log_directories"`
}

type DocumentOutput struct {
	Service  string `json:"service"`
	Path     string `json:"path"`
	Document string `json:"document"`
}

type InvestigationOutput struct {
	Query                   string                  `json:"query"`
	Service                 string                  `json:"service,omitempty"`
	Stage                   string                  `json:"stage"`
	ResolvedQuery           string                  `json:"resolved_query"`
	AttemptedQueries        []string                `json:"attempted_queries"`
	DocumentationMatchCount int                     `json:"documentation_match_count"`
	Result                  search.Result           `json:"result"`
	RelatedResults          []InvestigationEvidence `json:"related_results,omitempty"`
}

type InvestigationEvidence struct {
	Query  string        `json:"query"`
	Result search.Result `json:"result"`
}

func New(registry *services.Registry, reader *docs.Reader) *Server {
	return &Server{registry: registry, reader: reader}
}

func (s *Server) Register(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "list_services", Description: "List configured repositories and their local paths."}, s.listServices)
	mcp.AddTool(server, &mcp.Tool{Name: "get_service_overview", Description: "Read docs/service-overview.md for a configured service."}, s.getOverview)
	mcp.AddTool(server, &mcp.Tool{Name: "get_endpoints", Description: "Read docs/endpoints.md for a configured service."}, s.getEndpoints)
	mcp.AddTool(server, &mcp.Tool{Name: "get_integrations", Description: "Read docs/integrations.md for a configured service."}, s.getIntegrations)
	mcp.AddTool(server, &mcp.Tool{Name: "trace_workflow_logs", Description: "Search known keywords, symbols, endpoints, or correlation IDs in configured service code and log directories. Prefer investigate_service for natural-language questions; scope with service when known."}, s.trace)
	mcp.AddTool(server, &mcp.Tool{Name: "search_service_docs", Description: "Search documentation files first for a known keyword, heading, endpoint, or integration name. Scope with service when known."}, s.searchDocumentation)
	mcp.AddTool(server, &mcp.Tool{Name: "investigate_service", Description: "Investigate a natural-language question, concept, or code symbol. Searches scoped documentation first, then focused production code and logs only when documentation has no matches. Normalizes common identifier variants and returns at most 25 ranked evidence lines by default."}, s.investigate)
}

func (s *Server) listServices(context.Context, *mcp.CallToolRequest, EmptyInput) (*mcp.CallToolResult, ServiceOutput, error) {
	output := ServiceOutput{}
	for _, service := range s.registry.All() {
		output.Services = append(output.Services, ServiceSummary{Name: service.Name, RootPath: service.Root, LogDirectories: service.LogDirectories})
	}
	return toolResult(output, fmt.Sprintf("Found %d configured services.", len(output.Services)))
}

func (s *Server) getOverview(_ context.Context, _ *mcp.CallToolRequest, input ServiceInput) (*mcp.CallToolResult, DocumentOutput, error) {
	return s.readDocument(input.Service, "overview")
}

func (s *Server) getEndpoints(_ context.Context, _ *mcp.CallToolRequest, input ServiceInput) (*mcp.CallToolResult, DocumentOutput, error) {
	return s.readDocument(input.Service, "endpoints")
}

func (s *Server) getIntegrations(_ context.Context, _ *mcp.CallToolRequest, input ServiceInput) (*mcp.CallToolResult, DocumentOutput, error) {
	return s.readDocument(input.Service, "integrations")
}

func (s *Server) readDocument(serviceName, documentName string) (*mcp.CallToolResult, DocumentOutput, error) {
	service, err := s.registry.Get(serviceName)
	if err != nil {
		return nil, DocumentOutput{}, err
	}
	document, err := s.reader.Read(service, documentName)
	if err != nil {
		return nil, DocumentOutput{}, err
	}
	return toolResult(
		DocumentOutput{Service: document.Service, Path: document.Path, Document: document.Content},
		fmt.Sprintf("Read %s documentation for service %q.", documentName, document.Service),
	)
}

func (s *Server) trace(ctx context.Context, _ *mcp.CallToolRequest, input TraceInput) (*mcp.CallToolResult, search.Result, error) {
	serviceName := ""
	if input.Service != nil {
		serviceName = *input.Service
	}
	maxResults := 200
	if input.MaxResults != nil {
		maxResults = *input.MaxResults
		if maxResults < 1 || maxResults > search.MaxResultsLimit {
			return nil, search.Result{}, fmt.Errorf("max_results must be between 1 and %d", search.MaxResultsLimit)
		}
	}
	result, err := search.Trace(ctx, s.registry.All(), serviceName, input.Query, maxResults)
	if err != nil {
		return nil, search.Result{}, err
	}
	return toolResult(result, searchSummary("workflow", result))
}

func (s *Server) searchDocumentation(ctx context.Context, _ *mcp.CallToolRequest, input DocumentationSearchInput) (*mcp.CallToolResult, search.Result, error) {
	serviceName := ""
	if input.Service != nil {
		serviceName = *input.Service
	}
	maxResults := 200
	if input.MaxResults != nil {
		maxResults = *input.MaxResults
		if maxResults < 1 || maxResults > search.MaxResultsLimit {
			return nil, search.Result{}, fmt.Errorf("max_results must be between 1 and %d", search.MaxResultsLimit)
		}
	}
	result, err := search.TraceDocumentation(ctx, s.registry.All(), serviceName, input.Query, maxResults)
	if err != nil {
		return nil, search.Result{}, err
	}
	return toolResult(result, searchSummary("documentation", result))
}

func (s *Server) investigate(ctx context.Context, _ *mcp.CallToolRequest, input InvestigationInput) (*mcp.CallToolResult, InvestigationOutput, error) {
	serviceName := ""
	if input.Service != nil {
		serviceName = *input.Service
	}
	maxResults := 25
	if input.MaxResults != nil {
		maxResults = *input.MaxResults
	}
	if err := validateInvestigationInput(input.Query, maxResults); err != nil {
		return nil, InvestigationOutput{}, err
	}

	candidates := search.QueryCandidates(input.Query)
	output := InvestigationOutput{
		Query:            input.Query,
		Service:          serviceName,
		Stage:            "none",
		AttemptedQueries: make([]string, 0, len(candidates)),
	}
	var documentationEvidence []InvestigationEvidence
	for _, candidate := range candidates {
		result, err := search.TraceDocumentation(ctx, s.registry.All(), serviceName, candidate, maxResults)
		if err != nil {
			return nil, InvestigationOutput{}, err
		}
		output.AttemptedQueries = appendUnique(output.AttemptedQueries, candidate)
		output.DocumentationMatchCount += result.MatchCount
		if result.MatchCount > 0 {
			documentationEvidence = append(documentationEvidence, InvestigationEvidence{Query: candidate, Result: result})
		}
	}
	if len(documentationEvidence) > 0 {
		rankInvestigationEvidence(documentationEvidence)
		output.Stage = "documentation"
		output.ResolvedQuery = documentationEvidence[0].Query
		output.Result = documentationEvidence[0].Result
		output.RelatedResults = limitRelatedResults(documentationEvidence[1:])
		return toolResult(output, investigationSummary(output))
	}

	var workflowEvidence []InvestigationEvidence
	for _, candidate := range candidates {
		result, err := search.TraceFocused(ctx, s.registry.All(), serviceName, candidate, maxResults)
		if err != nil {
			return nil, InvestigationOutput{}, err
		}
		output.AttemptedQueries = appendUnique(output.AttemptedQueries, candidate)
		if result.MatchCount > 0 {
			workflowEvidence = append(workflowEvidence, InvestigationEvidence{Query: candidate, Result: result})
		}
	}
	if len(workflowEvidence) > 0 {
		rankInvestigationEvidence(workflowEvidence)
		output.Stage = "workflow"
		output.ResolvedQuery = workflowEvidence[0].Query
		output.Result = workflowEvidence[0].Result
		output.RelatedResults = limitRelatedResults(workflowEvidence[1:])
		return toolResult(output, investigationSummary(output))
	}

	if len(candidates) > 0 {
		output.ResolvedQuery = candidates[0]
	}
	return toolResult(output, investigationSummary(output))
}

func validateInvestigationInput(query string, maxResults int) error {
	if len(search.QueryCandidates(query)) == 0 {
		return fmt.Errorf("query must not be empty")
	}
	if maxResults < 1 || maxResults > search.MaxResultsLimit {
		return fmt.Errorf("max_results must be between 1 and %d", search.MaxResultsLimit)
	}
	return nil
}

func investigationSummary(output InvestigationOutput) string {
	switch output.Stage {
	case "documentation":
		if len(output.RelatedResults) > 0 {
			return fmt.Sprintf("Found documentation matches for %q and %q.", output.ResolvedQuery, output.RelatedResults[0].Query)
		}
		return fmt.Sprintf("Found %d documentation matches for %q.", output.Result.MatchCount, output.ResolvedQuery)
	case "workflow":
		if len(output.RelatedResults) > 0 {
			return fmt.Sprintf("No documentation matches; found focused workflow matches for %q and related evidence for %q.", output.ResolvedQuery, output.RelatedResults[0].Query)
		}
		return fmt.Sprintf("No documentation matches; found %d focused workflow matches for %q.", output.Result.MatchCount, output.ResolvedQuery)
	default:
		return fmt.Sprintf("No documentation or workflow matches found for %q.", output.Query)
	}
}

func limitRelatedResults(results []InvestigationEvidence) []InvestigationEvidence {
	if len(results) > 2 {
		results = results[:2]
	}
	for i := range results {
		if len(results[i].Result.Matches) > 10 {
			results[i].Result.Matches = results[i].Result.Matches[:10]
			results[i].Result.Truncated = true
		}
	}
	return results
}

func rankInvestigationEvidence(evidence []InvestigationEvidence) {
	sort.SliceStable(evidence, func(i, j int) bool {
		return investigationEvidenceScore(evidence[i]) > investigationEvidenceScore(evidence[j])
	})
}

func investigationEvidenceScore(evidence InvestigationEvidence) int {
	query := compactSearchText(evidence.Query)
	score := minInt(evidence.Result.MatchCount, 50)
	for _, match := range evidence.Result.Matches {
		if query != "" && strings.Contains(compactSearchText(match.File), query) {
			score += 1000
		}
	}
	return score
}

func compactSearchText(value string) string {
	var compact strings.Builder
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			compact.WriteRune(r)
		}
	}
	return compact.String()
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func toolResult[T any](value T, summary string) (*mcp.CallToolResult, T, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: summary}},
	}, value, nil
}

func searchSummary(kind string, result search.Result) string {
	if result.MatchCount == 0 {
		return fmt.Sprintf("No %s matches found for %q.", kind, result.Query)
	}
	if result.Truncated {
		return fmt.Sprintf("Found %d %s matches for %q; showing %d.", result.MatchCount, kind, result.Query, len(result.Matches))
	}
	return fmt.Sprintf("Found %d %s matches for %q.", result.MatchCount, kind, result.Query)
}
