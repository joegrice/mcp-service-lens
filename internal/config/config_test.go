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
	jsonData := `{"services":[{"name":"orders","root":"/tmp/orders","log_directories":["/tmp/orders/logs"]}]}`
	yamlData := "services:\n  - name: orders\n    root: /tmp/orders\n    log_directories:\n      - /tmp/orders/logs\n"
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
