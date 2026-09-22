# mcp-service-lens

A local [Model Context Protocol](https://modelcontextprotocol.io/) server for understanding standardized microservice documentation and tracing workflows across local repositories.

## Features

- Targeted reads of `docs/service-overview.md`, `docs/endpoints.md`, and `docs/integrations.md`
- YAML or JSON service configuration
- Concurrent code and log searches with `ripgrep`
- Stdio transport for local MCP clients
- Bounded, line-oriented results to protect model context

## Requirements

- Go 1.25+
- `rg` on `PATH`

## Usage

```sh
go run ./cmd/mcp-service-lens --config ./config.example.yaml
```

Set `MCP_SERVICE_LENS_CONFIG` instead of passing `--config` if preferred. Each service needs an absolute `root`; `log_directories` are optional.

Available tools:

- `list_services`
- `get_service_overview`
- `get_endpoints`
- `get_integrations`
- `trace_workflow_logs`

The server communicates over stdout using MCP. Diagnostics are written to stderr.

## Development

```sh
go test ./...
go vet ./...
```

Contributions are welcome. Please keep changes focused, add tests for behavior changes, and open an issue for larger design proposals. This project is licensed under the MIT License.
