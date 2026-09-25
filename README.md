# mcp-service-lens

A fast, local [Model Context Protocol](https://modelcontextprotocol.io/) server for understanding microservice documentation and tracing workflows across repositories.

Point it at a directory of local repositories and let your AI assistant find service contracts, integrations, and runtime evidence without manually searching each codebase.

## Genericity rule

Keep the MCP repository-agnostic. Do not hard-code service names, company-specific terminology, repository paths, integrations, or domain assumptions into the implementation, documentation, examples, or tests. Use neutral placeholder names and derive investigation behaviour from the configured repositories and their contents. Any new feature or regression test should work unchanged against an unrelated set of services.

## Install

Build the local binary:

```sh
go build -o ~/.local/bin/mcp-service-lens ./cmd/mcp-service-lens
```

Requires Go 1.25+ and [`ripgrep`](https://github.com/BurntSushi/ripgrep) on `PATH`.

## MCP Clients

Register `mcp-service-lens` as a local stdio MCP server in your client with:

```text
command: /path/to/mcp-service-lens
arguments: --directory /path/to/repositories
```

The MCP client owns the server process: it starts the binary when a session needs it and stops it when the session ends. No terminal window or long-running background process is required. The server communicates over stdin/stdout, so launching it directly in a terminal will appear to hang while it waits for MCP messages; that is expected. For development, use `go run` from the repository root instead.

### Codex CLI

Codex can register the local server globally and launch it automatically:

```sh
codex mcp add service-lens -- /path/to/mcp-service-lens --directory /path/to/repositories
codex mcp list
```

The same MCP configuration is available to the Codex CLI and IDE extension. Replace both paths with local absolute paths.

After registering it, ask your client questions such as:

```text
Find the documented request flow for an endpoint, then trace correlation ID req-123 across all services and identify the first failure.
```

## Investigation benchmark

In a local comparison using the same investigation, the MCP returned a focused documentation-first result while a manual `rg` search required broad results followed by narrowing:

| Method | Time | Returned text | Estimated tokens* |
| --- | ---: | ---: | ---: |
| MCP `investigate_service` | ~0.52s | 8.9 KB | ~2.2k |
| Manual `rg` + follow-up narrowing | ~1.47s | 224.8 KB | ~56k |

That run made the MCP approximately 2.8x faster with 25x less output/token volume. Token figures are estimates based on output characters divided by four, not provider billing telemetry; results vary with repository size and query.

## Capabilities

- Reads standard service overviews, endpoint contracts, and integration maps directly
- Searches arbitrary repository documentation case-insensitively with `search_service_docs`, optionally scoped to one service
- Traces correlation IDs, endpoint paths, and keywords through code and local logs, optionally scoped to one service
- Investigates natural-language questions with a documentation-first workflow and focused, ranked code fallback via `investigate_service`
- Returns concise human-readable summaries alongside full structured tool results
- Discovers repositories containing a `docs/` directory
- Supports explicit YAML or JSON configuration for custom layouts

See [development.md](docs/development.md) for contributor instructions and configuration details.

Contributions and issue reports are welcome. Licensed under the MIT License.
