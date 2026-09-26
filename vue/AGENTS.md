# Frontend instructions

Read and follow `../AGENTS.md` before changing frontend files.

## Implementation and debugging

- Follow the existing Vue 3, Vite, Pinia, and Vue Router patterns. Keep API access, state management, reusable components, and page views in their existing directories.
- Use `frontenddebugger` and Chrome DevTools MCP for frontend debugging, UI audits, and verification of changes affecting rendering, interaction, or API integration. Read the skill and relevant browser workflow before browser interaction.
- Start with the source and documented intent, reproduce the affected flow, inspect current accessibility snapshots, console messages, and network requests, and repeat the flow after the fix.
- Check relevant loading, empty, error, success, disabled, and authentication states. Verify responsive behavior or keyboard access when the change affects them.
- Save screenshots, snapshots, traces, generated photos, and temporary scripts in `<repository-root>/build/temp/` with descriptive unique names. Execute temporary scripts from that directory.
- Use `/perplexity-research` for any online search, website investigation, topic research, or framework/API lookup.
- Use descriptive JavaScript variables, function names, component names, and template bindings. Do not introduce single-character identifiers, meaningless abbreviations, or numbered placeholders.
- Replace unexplained business literals, status/provider values, timeouts, and limits with meaningful constants or configuration. Preserve established external API fields and valid JavaScript/Vue syntax.
- Keep LLM integration on the backend through Eino. Shared backend plugins belong in `src/internal/pkg/`; LLM-related plugins belong in `src/internal/pkg/eino/`.
- Keep frontend request/response handling aligned with backend routes and DTOs; inspect current configuration to determine service URLs.

## Validation and completion

- Run frontend commands from `vue/` using PowerShell, `cmd /c` for Windows wrappers where appropriate, or Git Bash for Bash scripts, with UTF-8 input/output.
- Available checks are `npm run lint:check` and `npm run build`. Run checks relevant to the edited files and report existing unrelated failures accurately.
- `npm run lint` rewrites files; review its scope before using it in a dirty workspace.
- `npm run dev` starts the development server on port 3000. Confirm the active server and backend proxy settings before browser verification.
- Do not report a runtime issue as fixed solely because lint or build passed; verify the affected browser flow using the required skill and tools.
- Finish with a root `devlog/YYYY-MM-DD/` entry describing changes, evidence, checks, and limitations, then commit and push only the task's files.
