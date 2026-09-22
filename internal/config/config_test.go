package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJSONAndYAML(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "config.json")
	yamlPath := filepath.Join(dir, "config.yaml")
	root := filepath.Join(dir, "orders")
	logs := filepath.Join(root, "logs")
	if err := os.MkdirAll(logs, 0700); err != nil {
		t.Fatal(err)
	}
	jsonData := `{"services":[{"name":"orders","root":"` + root + `","log_directories":["` + logs + `"]}]}`
	yamlData := "services:\n  - name: orders\n    root: " + root + "\n    log_directories:\n      - " + logs + "\n"
	if err := os.WriteFile(jsonPath, []byte(jsonData), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(yamlPath, []byte(yamlData), 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{jsonPath, yamlPath} {
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load(%q): %v", path, err)
		}
		if len(cfg.Services) != 1 || cfg.Services[0].Name != "orders" {
			t.Fatalf("unexpected config: %+v", cfg)
		}
	}
}

func TestValidateRejectsDuplicateAndRelativePaths(t *testing.T) {
	for name, cfg := range map[string]Config{
		"duplicate":     {Services: []Service{{Name: "orders", Root: "/tmp/orders"}, {Name: "orders", Root: "/tmp/other"}}},
		"relative root": {Services: []Service{{Name: "orders", Root: "orders"}}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := cfg.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidatePathsRejectsMissingRoot(t *testing.T) {
	err := (Config{Services: []Service{{Name: "orders", Root: filepath.Join(t.TempDir(), "missing")}}}).ValidatePaths()
	if err == nil {
		t.Fatal("expected missing root error")
	}
}

func TestDiscoverFindsDocumentedRepositoriesAndLogs(t *testing.T) {
	parent := t.TempDir()
	serviceRoot := filepath.Join(parent, "orders")
	if err := os.MkdirAll(filepath.Join(serviceRoot, "docs"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(serviceRoot, "documentation-generation-prompt.txt"), []byte("standard"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(serviceRoot, "logs"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(parent, "not-a-service"), 0700); err != nil {
		t.Fatal(err)
	}

	cfg, err := Discover(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Services) != 1 || cfg.Services[0].Name != "orders" {
		t.Fatalf("unexpected discovered services: %+v", cfg.Services)
	}
	if len(cfg.Services[0].LogDirectories) != 1 || cfg.Services[0].LogDirectories[0] != filepath.Join(serviceRoot, "logs") {
		t.Fatalf("unexpected discovered logs: %+v", cfg.Services[0].LogDirectories)
	}
}

func TestDiscoverAcceptsRepositoryWithGenericDocsDirectory(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "openwire")
	if err := os.MkdirAll(filepath.Join(root, "docs", "designs"), 0700); err != nil {
		t.Fatal(err)
	}

	cfg, err := Discover(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Services) != 1 || cfg.Services[0].Name != "openwire" {
		t.Fatalf("unexpected discovered services: %+v", cfg.Services)
	}
}
