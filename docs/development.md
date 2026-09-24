# Development

## Local Run

Run the stdio server from the repository root:

```sh
go run ./cmd/mcp-service-lens --directory /home/joe/Code
```

The process waits for an MCP client and will appear idle when launched directly in a terminal. Any MCP-compatible client can launch `mcp-service-lens` with `--directory /home/joe/Code`.

## Discovery

Directory mode scans immediate child directories containing `docs/`. It automatically detects these log locations when present:

- `logs/`
- `log/`
- `var/log/`

For custom layouts, use YAML or JSON configuration:

```sh
go run ./cmd/mcp-service-lens --config ./config.example.yaml
```

## Investigation workflow

Use [`investigation.md`](investigation.md) for the documentation-first workflow exposed by `investigate_service`. In short, provide a natural-language question and scope it to a configured service when possible. The tool searches documentation first, then returns focused and ranked code evidence plus related evidence groups for multi-concept questions.

## Verification

```sh
go test ./...
go vet ./...
```

Changes should include focused tests. Keep MCP stdout reserved for protocol messages; write diagnostics to stderr.
