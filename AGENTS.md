# Repository instructions

These rules apply to the entire repository. Also read the `AGENTS.md` in the directory being changed: `src/`, `docs/`, or `vue/`.

## Project layout

- `src/cmd/server/`: Go server entry point and application wiring.
- `src/internal/`: backend APIs, services, repositories, infrastructure, and shared packages; it has its own `go.mod`.
- `src/internal/pkg/`: shared plugin implementations and registration. In these instructions, `pkg` means this existing directory.
- `src/internal/pkg/eino/`: all LLM-related plugins, model adapters, research agents, and AI tools. `pkg/eino` means this directory.
- `vue/`: Vue 3 frontend using Vite, Pinia, and Vue Router.
- `docs/`: project documentation and local Eino references in `docs/eino/`.
- `devlog/YYYY-MM-DD/`: dated development logs for completed work.
- `build/temp/`: temporary scripts and generated debugging evidence.

## Required skills and tools

- Use the `frontenddebugger` skill together with Chrome DevTools MCP for frontend debugging, UI audits, and browser verification of frontend behavior or frontend-backend integration changes. Read the skill before starting the relevant work.
- Collect relevant accessibility snapshots, console messages, and network requests; reproduce the affected flow after a fix. Capture screenshots when visual evidence is needed.
- Use `/perplexity-research` (the `perplexity-research` skill) for every online search, website investigation, subject lookup, API documentation lookup, and web-based fact check. Read the skill and use its Perplexity tools; verify specific claims against primary sources or local source and include source links.
- If a required skill or tool is unavailable, report the limitation explicitly. Never silently substitute tools or claim browser verification or research that did not happen.
- Documentation-only changes do not require launching the browser or doing online research unless their content needs that evidence.

## Architecture and code clarity

- Use CloudWeGo Eino as the only LLM and agent framework. Build on Eino model, message, tool, streaming, and orchestration interfaces instead of adding another AI framework or a parallel custom LLM stack.
- Implement and register plugins under `src/internal/pkg/`. Implement and register LLM-related plugins under `src/internal/pkg/eino/`, using its existing model, tool, and agent packages as appropriate.
- Keep plugin implementation and registration out of API handlers, business services, and server entry points. Those layers consume the package's exported interfaces and initialization functions.
- Use descriptive identifiers. Do not introduce single-character names, meaningless abbreviations, numbered placeholder names, or names that hide their purpose.
- Do not introduce magic numbers, strings, or characters for business rules, timeouts, limits, status values, provider names, or protocol markers. Use meaningful named constants or configuration and document units where relevant.
- Keep text readable and correctly encoded. Avoid decorative symbols or unexplained character literals; preserve required language syntax and external API field names.
- Follow the existing handler/service/repository boundaries. Update frontend consumers and documentation when a backend contract changes.

## Shell, encoding, and temporary artifacts

- On Windows, use `exec_command` with `tty: true` by default to keep shell commands inside Codex's ConPTY and avoid visible console windows from the plain-pipe spawn path. Use `login: false` unless the task requires the user's PowerShell profile.
- For exact binary output, write to a file under `build/temp/` and read it rather than switching to plain-pipe execution.
- Use PowerShell for Windows commands and Git Bash for Bash scripts. Use `cmd /c` when a Windows command or wrapper requires it, including npm wrappers where appropriate.
- Use UTF-8 for terminal input/output and file reads/writes; prefer UTF-8 without BOM for source, Markdown, JSON, and scripts. In PowerShell, set `[Console]::InputEncoding`, `[Console]::OutputEncoding`, and `$OutputEncoding` to UTF-8 when needed, and use explicit encoding for file operations.
- Create `<repository-root>/build/temp/` if it does not exist before generating temporary artifacts.
- Save all temporary script files there and execute them from that directory, using explicit paths to project inputs when needed.
- Save screenshots, generated photos, browser snapshots, traces, temporary reports, and other debugging output under `build/temp/`. Use unique descriptive filenames to preserve earlier evidence.
- Keep temporary artifacts out of Git. Do not write temporary files into source, documentation, or development-log directories.
- Use native PowerShell file operations for deletion or moves; verify resolved paths before recursive operations.

## Validation and completion

- Inspect the existing Git status before editing and preserve unrelated work. Stage only files belonging to the current task.
- Run checks appropriate to the change. Backend changes may require checks in both Go modules; frontend behavior changes require the skill and browser checks above in addition to relevant lint/build checks.
- After each completed task, add or update a descriptive Markdown log in `devlog/YYYY-MM-DD/`, using the local calendar date. The date format is year-month-day, for example `devlog/2026-09-26/`.
- Record the request, changes, validation results, and remaining limitations. Reference evidence in `build/temp/` by path rather than copying temporary artifacts into the log.
- Review the staged diff and run `git diff --cached --check`, then create a descriptive Git commit containing the task changes and its development log.
- Push the commit to the current branch's configured upstream. Use a normal push; do not force-push. If there is no upstream, push the current branch to the configured project remote with upstream tracking.
- Commit and push are part of completion and already requested. If either fails, report the concrete failure and leave local work intact; never claim a successful push without verification.
