# Backend instructions

Read and follow `../AGENTS.md` before changing backend files.

## Implementation

- Keep HTTP routing and validation in API/handler packages, business behavior in services, and database access in repository/DAO packages.
- Use Eino exclusively for LLM calls, messages, streaming, agents, and research orchestration. Consult `../docs/eino/` before changing these integrations; use `/perplexity-research` for any needed online investigation.
- Add shared plugins and their registration in `internal/pkg/`. Add every LLM-related plugin and its registration in `internal/pkg/eino/`; use its `model/`, `tool/`, and `agent/` packages where appropriate.
- Export package initialization functions for the server to call. Do not add provider or tool implementations or registration logic to `cmd/server/`, handlers, or services.
- Avoid extending legacy custom LLM abstractions in `internal/pkg/llm/` or `internal/pkg/tools/` for new AI integrations; use Eino interfaces under `internal/pkg/eino/`.
- Preserve cancellation, error propagation, streaming cleanup, and existing API behavior when changing model or tool integrations.
- Give variables, parameters, receivers, functions, and constants descriptive names. Do not introduce single-character or meaningless identifiers or unexplained numeric/string/character literals.
- Update Vue API consumers and relevant documentation when request, response, authentication, or streaming contracts change. Verify affected UI flows with `frontenddebugger` and Chrome DevTools MCP.

## Validation

- The root module replaces `github.com/ai-research-platform/internal` with `./src/internal`. Check the appropriate module rather than assuming one root test command covers both.
- Format changed Go files with `gofmt`. Run focused tests for changed packages; run broader checks when the scope requires them.
- Run server checks from the repository root, for example `go test ./src/cmd/server`.
- Run internal package checks from `src/internal/`, for example `go test ./pkg/eino/...`; `go test ./...` there covers the internal module when a full suite is appropriate.
- Run database integration tests only with an explicitly isolated test database. Report skipped checks or failures accurately.
- Use UTF-8 for Go source, scripts, and command input/output. Store temporary test scripts and generated evidence in the repository root's `build/temp/`.
- Finish with a dated root `devlog/YYYY-MM-DD/` entry, a scoped commit, and a normal push as required by the root instructions.
