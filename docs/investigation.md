# Service Investigation

Use `investigate_service` for natural-language questions that may require both documentation and code evidence.

## Workflow

The tool follows this order:

1. Searches documentation for the configured service.
2. Normalises likely identifier variants, such as `LegacyServiceCoordinator` to `ServiceCoordinator`.
3. Falls back to focused production-code and log searches when documentation has no matches.
4. Returns the primary evidence group plus up to two related groups when the question contains multiple concepts.

Focused searches return a maximum of 25 evidence lines by default, rank likely definitions and call sites first, and exclude test and infrastructure paths. Use `max_results` to change the limit when necessary.

## Input

```json
{
  "query": "what metadata does ServiceCoordinator call MetadataProvider for?",
  "service": "example-service",
  "max_results": 10
}
```

`service` is optional, but providing it is recommended. It reduces search time, result noise, and token usage.

## Result fields

- `stage`: `documentation`, `workflow`, or `none`.
- `resolved_query`: the repository term that produced the primary evidence.
- `attempted_queries`: the bounded list of normalised terms that were searched.
- `documentation_match_count`: documentation matches found before code fallback.
- `result`: the primary evidence group.
- `related_results`: additional evidence groups for other concepts in the question.

For example, a ServiceCoordinator/MetadataProvider investigation should return a primary `ServiceCoordinator` group and a related `MetadataProvider` group. The result groups are evidence, not a guaranteed call graph; inspect the referenced files and lines before claiming a direct dependency.

## Choosing another tool

- Use `get_service_overview`, `get_endpoints`, or `get_integrations` when the document is known.
- Use `search_service_docs` for a direct documentation keyword or heading search.
- Use `trace_workflow_logs` for an exact correlation ID, endpoint, log message, or known code symbol.
- Use `investigate_service` for questions that need discovery, normalisation, or multiple related concepts.
