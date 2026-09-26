# Documentation instructions

Read and follow `../AGENTS.md` before changing documentation.

- Check current source, configuration, and tests before describing behavior; distinguish implemented behavior, planned work, and known limitations.
- Use `/perplexity-research` for online searches, website research, subject research, and external documentation lookup. Include primary-source links and identify unverified claims.
- Document Eino as the sole LLM/agent framework. Use `docs/eino/` as local reference material and confirm details against current dependencies and source.
- Document plugin implementation and registration under `src/internal/pkg/`, with LLM-related plugins under `src/internal/pkg/eino/`.
- Use clear, descriptive terminology, UTF-8 text, and meaningful example names. Explain configuration values and units; avoid magic literals and decorative characters.
- Check local links and example paths. Do not present stale documentation as proof that the application currently behaves as described.
- UI debugging or browser-based evidence collection must use `frontenddebugger` and Chrome DevTools MCP. Ordinary documentation-only edits do not require a browser session.
- Save temporary scripts, screenshots, generated photos, and research/debugging artifacts in root `build/temp/`; link to evidence rather than embedding temporary files in this directory.
- After completed work, write its log under root `devlog/YYYY-MM-DD/`, then commit and push the scoped changes.
- The repository ignores new files under `docs/` except this instruction file. When new documentation belongs to the task, explicitly stage only those intended files with `git add -f -- <path>`; do not force-add the whole directory.
