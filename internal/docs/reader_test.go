package docs

import (
	"os"
	"path/filepath"
	"testing"

	"mcp-service-lens/internal/config"
)

func TestReaderReadsAndCachesTargetedDocument(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	if err := os.Mkdir(docsDir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(docsDir, "endpoints.md")
	if err := os.WriteFile(path, []byte("# Endpoints\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r := NewReader()
	document, err := r.Read(config.Service{Name: "orders", Root: root}, "endpoints")
	if err != nil {
		t.Fatal(err)
	}
	if document.Content != "# Endpoints\n" || document.Path != path {
		t.Fatalf("unexpected document: %+v", document)
	}
}
