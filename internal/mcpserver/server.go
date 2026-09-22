package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

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
	Query      string `json:"query" jsonschema:"correlation ID, keyword, or endpoint path"`
	MaxResults *int   `json:"max_results,omitempty" jsonschema:"optional maximum number of matching lines to return; must be between 1 and 1000"`
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

func New(registry *services.Registry, reader *docs.Reader) *Server {
	return &Server{registry: registry, reader: reader}
}

func (s *Server) Register(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{Name: "list_services", Description: "List configured repositories and their local paths."}, s.listServices)
	mcp.AddTool(server, &mcp.Tool{Name: "get_service_overview", Description: "Read docs/service-overview.md for a configured service."}, s.getOverview)
	mcp.AddTool(server, &mcp.Tool{Name: "get_endpoints", Description: "Read docs/endpoints.md for a configured service."}, s.getEndpoints)
	mcp.AddTool(server, &mcp.Tool{Name: "get_integrations", Description: "Read docs/integrations.md for a configured service."}, s.getIntegrations)
	mcp.AddTool(server, &mcp.Tool{Name: "trace_workflow_logs", Description: "Search configured service code and log directories with ripgrep."}, s.trace)
}

func (s *Server) listServices(context.Context, *mcp.CallToolRequest, EmptyInput) (*mcp.CallToolResult, ServiceOutput, error) {
	output := ServiceOutput{}
	for _, service := range s.registry.All() {
		output.Services = append(output.Services, ServiceSummary{Name: service.Name, RootPath: service.Root, LogDirectories: service.LogDirectories})
	}
	return textResult(output)
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
	return textResult(DocumentOutput{Service: document.Service, Path: document.Path, Document: document.Content})
}

func (s *Server) trace(ctx context.Context, _ *mcp.CallToolRequest, input TraceInput) (*mcp.CallToolResult, search.Result, error) {
	maxResults := 200
	if input.MaxResults != nil {
		maxResults = *input.MaxResults
		if maxResults < 1 || maxResults > search.MaxResultsLimit {
			return nil, search.Result{}, fmt.Errorf("max_results must be between 1 and %d", search.MaxResultsLimit)
		}
	}
	result, err := search.Trace(ctx, s.registry.All(), input.Query, maxResults)
	if err != nil {
		return nil, search.Result{}, err
	}
	return textResult(result)
}

func textResult[T any](value T) (*mcp.CallToolResult, T, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, value, fmt.Errorf("encode tool result: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(data)}}}, value, nil
}
