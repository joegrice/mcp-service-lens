package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Services []Service `json:"services" yaml:"services"`
}

type Service struct {
	Name           string   `json:"name" yaml:"name"`
	Root           string   `json:"root" yaml:"root"`
	LogDirectories []string `json:"log_directories" yaml:"log_directories"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	var cfg Config
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		err = json.Unmarshal(data, &cfg)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &cfg)
	default:
		return Config{}, fmt.Errorf("unsupported config extension %q; use .json, .yaml, or .yml", filepath.Ext(path))
	}
	if err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config %q: %w", path, err)
	}
	if err := cfg.ValidatePaths(); err != nil {
		return Config{}, fmt.Errorf("validate paths in config %q: %w", path, err)
	}
	return cfg, nil
}

// Discover finds documented repositories directly below parent.
func Discover(parent string) (Config, error) {
	info, err := os.Stat(parent)
	if err != nil {
		return Config{}, fmt.Errorf("stat discovery directory %q: %w", parent, err)
	}
	if !info.IsDir() {
		return Config{}, fmt.Errorf("discovery path %q is not a directory", parent)
	}

	entries, err := os.ReadDir(parent)
	if err != nil {
		return Config{}, fmt.Errorf("read discovery directory %q: %w", parent, err)
	}
	var services []Service
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		root := filepath.Join(parent, entry.Name())
		entryInfo, err := os.Stat(root)
		if err != nil || !entryInfo.IsDir() || !hasDocumentation(root) {
			continue
		}
		services = append(services, Service{
			Name:           entry.Name(),
			Root:           root,
			LogDirectories: discoverLogDirectories(root),
		})
	}
	if len(services) == 0 {
		return Config{}, fmt.Errorf("no documented repositories found directly under %q", parent)
	}
	cfg := Config{Services: services}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, cfg.ValidatePaths()
}

func hasDocumentation(root string) bool {
	if fileExists(filepath.Join(root, "documentation-generation-prompt.txt")) {
		return true
	}
	for _, file := range []string{"service-overview.md", "endpoints.md", "integrations.md"} {
		if fileExists(filepath.Join(root, "docs", file)) {
			return true
		}
	}
	return false
}

func discoverLogDirectories(root string) []string {
	var directories []string
	for _, relative := range []string{"logs", "log", filepath.Join("var", "log")} {
		path := filepath.Join(root, relative)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			directories = append(directories, path)
		}
	}
	return directories
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func (c Config) Validate() error {
	if len(c.Services) == 0 {
		return errors.New("services must not be empty")
	}
	seen := make(map[string]struct{}, len(c.Services))
	for i := range c.Services {
		s := &c.Services[i]
		if strings.TrimSpace(s.Name) == "" {
			return fmt.Errorf("services[%d].name must not be empty", i)
		}
		if _, ok := seen[s.Name]; ok {
			return fmt.Errorf("duplicate service name %q", s.Name)
		}
		seen[s.Name] = struct{}{}
		if !filepath.IsAbs(s.Root) {
			return fmt.Errorf("service %q root must be absolute", s.Name)
		}
		for _, dir := range s.LogDirectories {
			if !filepath.IsAbs(dir) {
				return fmt.Errorf("service %q log directory must be absolute: %q", s.Name, dir)
			}
		}
	}
	return nil
}

func (c Config) ValidatePaths() error {
	for _, service := range c.Services {
		if info, err := os.Stat(service.Root); err != nil || !info.IsDir() {
			if err != nil {
				return fmt.Errorf("service %q root %q: %w", service.Name, service.Root, err)
			}
			return fmt.Errorf("service %q root %q is not a directory", service.Name, service.Root)
		}
		for _, dir := range service.LogDirectories {
			if info, err := os.Stat(dir); err != nil || !info.IsDir() {
				if err != nil {
					return fmt.Errorf("service %q log directory %q: %w", service.Name, dir, err)
				}
				return fmt.Errorf("service %q log directory %q is not a directory", service.Name, dir)
			}
		}
	}
	return nil
}
