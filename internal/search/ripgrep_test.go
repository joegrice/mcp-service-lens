package search

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"mcp-service-lens/internal/config"
)

func TestTraceSearchesCodeAndLogs(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep is not installed")
	}
	root := t.TempDir()
	logs := filepath.Join(root, "logs")
	if err := os.Mkdir(logs, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "client.go"), []byte("call /payments\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "dist"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "dist", "bundle.js"), []byte("call /payments\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logs, "app.log"), []byte("request /payments\n"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := Trace(context.Background(), []config.Service{{Name: "orders", Root: root, LogDirectories: []string{logs}}}, "/payments", 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.MatchCount != 2 || len(result.Matches) != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Matches[0].Source != "code" || result.Matches[1].Source != "log" {
		t.Fatalf("matches are not deterministically ordered: %+v", result.Matches)
	}
}

func TestSortMatchesUsesStableServiceSourceFileLineOrder(t *testing.T) {
	matches := []Match{
		{Service: "payments", Source: "log", File: "/tmp/z.log", Line: 2},
		{Service: "orders", Source: "code", File: "/tmp/b.go", Line: 4},
		{Service: "orders", Source: "code", File: "/tmp/a.go", Line: 2},
	}
	sortMatches(matches)
	if matches[0].File != "/tmp/a.go" || matches[1].File != "/tmp/b.go" || matches[2].Service != "payments" {
		t.Fatalf("unexpected match order: %+v", matches)
	}
}

func TestTraceRejectsInvalidResultLimit(t *testing.T) {
	for _, limit := range []int{0, MaxResultsLimit + 1} {
		if _, err := Trace(context.Background(), nil, "query", limit); err == nil {
			t.Fatalf("expected error for max results %d", limit)
		}
	}
}

func TestTraceDocumentationSearchesGenericDocs(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep is not installed")
	}
	root := t.TempDir()
	docs := filepath.Join(root, "docs", "designs")
	if err := os.MkdirAll(docs, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docs, "architecture.md"), []byte("IGDB integration\n"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := TraceDocumentation(context.Background(), []config.Service{{Name: "openwire", Root: root}}, "", "IGDB", 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.MatchCount != 1 || len(result.Matches) != 1 || result.Matches[0].Source != "documentation" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
