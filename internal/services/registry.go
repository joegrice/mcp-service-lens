package services

import (
	"fmt"

	"mcp-service-lens/internal/config"
)

type Registry struct {
	services []config.Service
}

func New(cfg config.Config) *Registry { return &Registry{services: cfg.Services} }

func (r *Registry) All() []config.Service {
	result := make([]config.Service, len(r.services))
	copy(result, r.services)
	return result
}

func (r *Registry) Get(name string) (config.Service, error) {
	for _, service := range r.services {
		if service.Name == name {
			return service, nil
		}
	}
	return config.Service{}, fmt.Errorf("unknown service %q", name)
}
