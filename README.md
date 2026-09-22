# mcp-service-lens

A local [Model Context Protocol](https://modelcontextprotocol.io/) server for understanding standardized microservice documentation and tracing workflows across local repositories.

## Features

- Targeted reads of `docs/service-overview.md`, `docs/endpoints.md`, and `docs/integrations.md`
- Automatic discovery of documented repositories below a selected directory
- Generic documentation search for repositories without the standard filenames
- YAML or JSON service configuration
- Concurrent code and log searches with `ripgrep`
- Stdio transport for local MCP clients
- Bounded, line-oriented results to protect model context

## Requirements

- Go 1.25+
- `rg` on `PATH`

## Usage

```sh
go run ./cmd/mcp-service-lens --directory /home/joe/Code
```

Directory mode discovers immediate child repositories containing a `docs/` directory. It also detects `logs`, `log`, and `var/log` directories.

This starts an MCP stdio process and waits for an MCP client, so running it directly in a terminal will appear idle. For OpenCode, use the installed binary:

```json
{
  "mcp": {
    "service-lens": {
      "type": "local",
      "command": [
        "/home/joe/.local/bin/mcp-service-lens",
        "--directory",
        "/home/joe/Code"
      ],
      "enabled": true
    }
  }
}
```

Use `--config ./config.example.yaml` or `MCP_SERVICE_LENS_CONFIG` for explicit service and log mappings. Use `MCP_SERVICE_LENS_DIRECTORY` instead of passing `--directory` if preferred.

Available tools:

- `list_services`
- `get_service_overview`
- `get_endpoints`
- `get_integrations`
- `trace_workflow_logs`
- `search_service_docs`

`search_service_docs` is the fallback for repositories whose documentation does not use the standard filenames. For example:

```text
Search all service documentation for "IGDB" and summarize the relevant design and integration notes.
```

The server communicates over stdout using MCP. Diagnostics are written to stderr.

## Development

```sh
go test ./...
go vet ./...
```

Contributions are welcome. Please keep changes focused, add tests for behavior changes, and open an issue for larger design proposals. This project is licensed under the MIT License.
