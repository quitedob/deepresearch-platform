# Initialize repository agent rules

Date: 2026-09-26 (Asia/Shanghai)

## Request

Initialize instructions for `src/`, `docs/`, and `vue/`: require Chrome DevTools and `frontenddebugger` for frontend debugging, `/perplexity-research` for online investigation, Eino as the only AI framework, meaningful identifiers and named constants, package-based plugin registration, UTF-8 shell/file handling, dated logs, commits and pushes, and temporary artifacts under `build/temp/`.

## Changes

- Added root `AGENTS.md` and scoped instructions in `src/AGENTS.md`, `docs/AGENTS.md`, and `vue/AGENTS.md`.
- Mapped `pkg` and `pkg/eino` to the existing `src/internal/pkg/` and `src/internal/pkg/eino/` directories, including implementation and registration requirements.
- Recorded the required skill/tool workflows, code naming rules, Eino-only architecture, PowerShell / `cmd /c` / Git Bash usage, and UTF-8 encoding.
- Incorporated the follow-up Windows execution requirement: `exec_command` defaults to `tty: true` and `login: false`; exact binary output is written to a file and read back rather than using plain pipes.
- Established root `devlog/YYYY-MM-DD/` entries after completed work and scoped Git commits followed by normal upstream pushes. Interpreted the requested date pattern as calendar year-month-day.
- Created local `build/temp/` for temporary scripts, screenshots, generated photos, and other debugging evidence; its contents remain ignored by Git.
- Narrowed the documentation ignore rule to allow tracking `docs/AGENTS.md` while preserving the ignore behavior for other new documentation.
- Preserved the pre-existing backend/frontend changes and untracked files; they are outside this task's commit scope.

## Validation

- Inspected repository layout, both Go module files, frontend package scripts, existing documentation, plugin interfaces, branch/upstream, and initial Git status.
- Checked all four instruction files for valid UTF-8 without BOM and coverage of the required Eino, debugging, research, artifact, and development-log rules.
- Confirmed the referenced project/package paths and created directories exist.
- Confirmed `docs/AGENTS.md` is trackable and `build/temp/` artifacts and unrelated new documentation remain ignored.
- Reviewed the scoped changes and checked whitespace before committing.
- No application code changed. Backend tests, frontend build, browser debugging, and online research are outside this documentation-only task.

## Delivery

The task files and this log are to be committed on `feat/durable-research-submission` and pushed to its configured upstream, `origin/feat/durable-research-submission`. The final response reports the actual commit and push result.
