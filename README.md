# mcp-service-lens

A fast, local [Model Context Protocol](https://modelcontextprotocol.io/) server for understanding microservice documentation and tracing workflows across repositories.

Point it at a directory of local repositories and let your AI assistant find service contracts, integrations, and runtime evidence without manually searching each codebase.

## Install

Build the local binary:

```sh
go build -o ~/.local/bin/mcp-service-lens ./cmd/mcp-service-lens
```

Requires Go 1.25+ and [`ripgrep`](https://github.com/BurntSushi/ripgrep) on `PATH`.

## MCP Clients

Register `mcp-service-lens` as a local stdio MCP server in your client with:

```text
command: /home/joe/.local/bin/mcp-service-lens
arguments: --directory /home/joe/Code
```

For development, use `go run` from the repository root instead. The server communicates over stdin/stdout and should be launched by the MCP client, not used as an interactive terminal command.

After registering it, ask your client questions such as:

```text
Find the documented request flow for the checkout endpoint, then trace correlation ID req-123 across all services and identify the first failure.
```

## Capabilities

- Reads standard service overviews, endpoint contracts, and integration maps directly
- Searches arbitrary repository documentation with `search_service_docs`
- Traces correlation IDs, endpoint paths, and keywords through code and local logs
- Discovers repositories containing a `docs/` directory
- Supports explicit YAML or JSON configuration for custom layouts

See [development.md](docs/development.md) for contributor instructions and configuration details.

Contributions and issue reports are welcome. Licensed under the MIT License.
