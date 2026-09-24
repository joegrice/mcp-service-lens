package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"mcp-service-lens/internal/config"
	"mcp-service-lens/internal/docs"
	"mcp-service-lens/internal/search"
	"mcp-service-lens/internal/services"
)

func TestToolResultUsesConciseContent(t *testing.T) {
	result, output, err := toolResult(struct{ Value string }{Value: "full structured value"}, "compact summary")
	if err != nil {
		t.Fatal(err)
	}
	if output.Value != "full structured value" {
		t.Fatalf("structured output was changed: %+v", output)
	}
	if len(result.Content) != 1 {
		t.Fatalf("unexpected content: %+v", result.Content)
	}
	content, ok := result.Content[0].(*mcp.TextContent)
	if !ok || content.Text != "compact summary" {
		t.Fatalf("unexpected summary content: %+v", result.Content[0])
	}
}

func TestSearchSummary(t *testing.T) {
	tests := []struct {
		name   string
		kind   string
		result search.Result
		want   string
	}{
		{
			name:   "no matches",
			kind:   "documentation",
			result: search.Result{Query: "missing"},
			want:   `No documentation matches found for "missing".`,
		},
		{
			name:   "complete",
			kind:   "workflow",
			result: search.Result{Query: "needle", MatchCount: 2, Matches: make([]search.Match, 2)},
			want:   `Found 2 workflow matches for "needle".`,
		},
		{
			name:   "truncated",
			kind:   "workflow",
			result: search.Result{Query: "needle", MatchCount: 20, Matches: make([]search.Match, 5), Truncated: true},
			want:   `Found 20 workflow matches for "needle"; showing 5.`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := searchSummary(test.kind, test.result); got != test.want {
				t.Fatalf("summary = %q, want %q", got, test.want)
			}
		})
	}
}

func TestInvestigateSearchesDocumentationBeforeWorkflow(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "integrations.md"), []byte("ServiceCoordinator integration\n"), 0600); err != nil {
		t.Fatal(err)
	}

	server := New(services.New(config.Config{Services: []config.Service{{Name: "orders", Root: root}}}), docs.NewReader())
	result, output, err := server.investigate(context.Background(), nil, InvestigationInput{Query: "ServiceCoordinator", MaxResults: intPtr(5)})
	if err != nil {
		t.Fatal(err)
	}
	if output.Stage != "documentation" || output.ResolvedQuery != "ServiceCoordinator" {
		t.Fatalf("unexpected investigation output: %+v", output)
	}
	if output.Result.MatchCount != 1 || output.Result.Matches[0].Source != "documentation" {
		t.Fatalf("unexpected investigation result: %+v", output.Result)
	}
	if len(result.Content) != 1 {
		t.Fatalf("unexpected content: %+v", result.Content)
	}
}

func TestInvestigateNormalizesAndFallsBackToFocusedWorkflowSearch(t *testing.T) {
	root := t.TempDir()
	for _, directory := range []string{"docs", "src", "tests"} {
		if err := os.MkdirAll(filepath.Join(root, directory), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "src", "ServiceCoordinator.cs"), []byte("class ServiceCoordinator {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tests", "ServiceCoordinatorTests.cs"), []byte("ServiceCoordinator\n"), 0600); err != nil {
		t.Fatal(err)
	}

	server := New(services.New(config.Config{Services: []config.Service{{Name: "orders", Root: root}}}), docs.NewReader())
	_, output, err := server.investigate(context.Background(), nil, InvestigationInput{Query: "what does LegacyServiceCoordinator call?", MaxResults: intPtr(5)})
	if err != nil {
		t.Fatal(err)
	}
	if output.Stage != "workflow" || output.ResolvedQuery != "ServiceCoordinator" {
		t.Fatalf("unexpected investigation output: %+v", output)
	}
	if output.DocumentationMatchCount != 0 || output.Result.MatchCount != 1 || len(output.Result.Matches) != 1 {
		t.Fatalf("unexpected fallback result: %+v", output)
	}
	if output.Result.Matches[0].File != filepath.Join(root, "src", "ServiceCoordinator.cs") {
		t.Fatalf("focused search returned the wrong evidence: %+v", output.Result.Matches)
	}
}

func TestInvestigateReturnsRelatedEvidenceGroups(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0700); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "src")
	if err := os.MkdirAll(src, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "ServiceCoordinator.cs"), []byte("class ServiceCoordinator {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "MetadataProvider.cs"), []byte("class MetadataProvider {}\n"), 0600); err != nil {
		t.Fatal(err)
	}

	server := New(services.New(config.Config{Services: []config.Service{{Name: "orders", Root: root}}}), docs.NewReader())
	_, output, err := server.investigate(context.Background(), nil, InvestigationInput{Query: "what metadata service coordinator calls metadata provider for", MaxResults: intPtr(5)})
	if err != nil {
		t.Fatal(err)
	}
	if output.Stage != "workflow" || output.ResolvedQuery != "ServiceCoordinator" {
		t.Fatalf("unexpected investigation output: %+v", output)
	}
	if len(output.RelatedResults) != 1 || output.RelatedResults[0].Query != "MetadataProvider" {
		t.Fatalf("related evidence was not retained: %+v", output.RelatedResults)
	}
}

func intPtr(value int) *int { return &value }
