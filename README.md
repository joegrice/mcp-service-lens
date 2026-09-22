# mcp-service-lens

A fast, local [Model Context Protocol](https://modelcontextprotocol.io/) server for understanding microservice documentation and tracing workflows across repositories.

Point it at a directory of local repositories and let your AI assistant find service contracts, integrations, and runtime evidence without manually searching each codebase.

## Install

Build the local binary:

```sh
go build -o ~/.local/bin/mcp-service-lens ./cmd/mcp-service-lens
```

Requires Go 1.25+ and [`ripgrep`](https://github.com/BurntSushi/ripgrep) on `PATH`.

## OpenCode

Add the server to `~/.config/opencode/opencode.jsonc`:

```json
{
  "$schema": "https://opencode.ai/config.json",
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

Restart OpenCode, then ask questions such as:

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
