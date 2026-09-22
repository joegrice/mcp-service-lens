package docs

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"mcp-service-lens/internal/config"
)

type Reader struct {
	mu    sync.Mutex
	cache map[string]cachedFile
}

type cachedFile struct {
	modTime int64
	size    int64
	data    string
}

type Document struct {
	Service string `json:"service"`
	Path    string `json:"path"`
	Content string `json:"document"`
}

func NewReader() *Reader { return &Reader{cache: make(map[string]cachedFile)} }

func (r *Reader) Read(service config.Service, name string) (Document, error) {
	allowed := map[string]string{
		"overview":     "service-overview.md",
		"endpoints":    "endpoints.md",
		"integrations": "integrations.md",
	}
	file, ok := allowed[name]
	if !ok {
		return Document{}, fmt.Errorf("unknown document %q", name)
	}
	path := filepath.Join(service.Root, "docs", file)
	info, err := os.Stat(path)
	if err != nil {
		return Document{}, fmt.Errorf("stat %s: %w", path, err)
	}

	r.mu.Lock()
	if cached, ok := r.cache[path]; ok && cached.modTime == info.ModTime().UnixNano() && cached.size == info.Size() {
		r.mu.Unlock()
		return Document{Service: service.Name, Path: path, Content: cached.data}, nil
	}
	r.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return Document{}, fmt.Errorf("read %s: %w", path, err)
	}
	r.mu.Lock()
	r.cache[path] = cachedFile{modTime: info.ModTime().UnixNano(), size: info.Size(), data: string(data)}
	r.mu.Unlock()
	return Document{Service: service.Name, Path: path, Content: string(data)}, nil
}
