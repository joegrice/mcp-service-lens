# Development

## Local Run

Run the stdio server from the repository root:

```sh
go run ./cmd/mcp-service-lens --directory /home/joe/Code
```

The process waits for an MCP client and will appear idle when launched directly in a terminal.

## Discovery

Directory mode scans immediate child directories containing `docs/`. It automatically detects these log locations when present:

- `logs/`
- `log/`
- `var/log/`

For custom layouts, use YAML or JSON configuration:

```sh
go run ./cmd/mcp-service-lens --config ./config.example.yaml
```

## Verification

```sh
go test ./...
go vet ./...
```

Changes should include focused tests. Keep MCP stdout reserved for protocol messages; write diagnostics to stderr.
