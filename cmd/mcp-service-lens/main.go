package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"mcp-service-lens/internal/config"
	"mcp-service-lens/internal/docs"
	"mcp-service-lens/internal/mcpserver"
	"mcp-service-lens/internal/services"
)

func main() {
	configPath := flag.String("config", os.Getenv("MCP_SERVICE_LENS_CONFIG"), "path to a .json, .yaml, or .yml configuration file")
	directory := flag.String("directory", os.Getenv("MCP_SERVICE_LENS_DIRECTORY"), "parent directory containing documented repositories")
	flag.Parse()
	if *configPath != "" && *directory != "" {
		log.Fatal("use either --config or --directory, not both")
	}
	if *configPath == "" && *directory == "" {
		log.Fatal("configuration is required; use --config, --directory, MCP_SERVICE_LENS_CONFIG, or MCP_SERVICE_LENS_DIRECTORY")
	}

	var (
		cfg config.Config
		err error
	)
	if *directory != "" {
		cfg, err = config.Discover(*directory)
	} else {
		cfg, err = config.Load(*configPath)
	}
	if err != nil {
		log.Fatal(err)
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "mcp-service-lens", Version: "0.1.0"}, nil)
	mcpserver.New(services.New(cfg), docs.NewReader()).Register(server)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
