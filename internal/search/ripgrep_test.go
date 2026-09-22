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
}
