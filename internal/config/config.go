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
	return cfg, nil
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
