---
name: skm-maintainer
description: Maintain the skm Go project and its AI Skill Manager workflows. Use when implementing, reviewing, testing, or documenting changes in this repository, especially CLI, macOS Core Bridge, project deployment, Codex/Claude skill activation, or YAML state.
---

# SKM Maintainer

Use this skill for changes to the `skm` repository. Preserve the existing separation between Library, Activation, and Project state. The supported interfaces are the Go CLI and the native macOS App, which calls the bundled Go Core through private stdio JSON-RPC.

## Workflow

1. Inspect the relevant package, tests, docs, and current git status before editing. Preserve unrelated user changes.
2. Trace the behavior through its ownership boundary. Keep CLI parsing in `internal/cli`, use-case orchestration in `internal/application`, native protocol handling in `internal/corebridge`, persistence in `internal/store`, domain rules in `internal/domain` or `internal/planner`, and skill validation in `internal/skill`. Keep macOS presentation in `macos/`; Swift must not duplicate Go storage or deployment rules.
3. Make the smallest complete change. Keep project-portable state under `.skm/`; keep Codex-discoverable repository skills under `.agents/skills/`.
4. Add or update focused tests for changed behavior. Use table-driven tests where the package already does so.
5. Run `gofmt` on changed Go files, then run `go test ./...` and `go build ./...`. For CLI or Core changes, also run `sh scripts/dev_smoke_test.sh`. For native App changes, use the build and test commands in `macos/README.md` with isolated test roots.
6. Report changed files, verification commands, and any remaining limitation. Do not claim a test passed unless it was run.

## Skill Changes

When creating or updating a skill:

- Keep `SKILL.md` frontmatter limited to valid metadata with a lowercase hyphenated `name` and a specific trigger-oriented `description`.
- Add `agents/openai.yaml` with quoted UI strings: `display_name`, a 25-64 character `short_description`, and a one-sentence `default_prompt` that names `$skill-name`.
- Add `scripts/`, `references/`, or `assets/` only when they directly support the workflow.
- Validate the skill with `python3 "${CODEX_HOME:-$HOME/.codex}/skills/.system/skill-creator/scripts/quick_validate.py" <skill-directory>`.

## Boundaries

- Never place credentials, machine-specific home paths, or personal Codex state in the repository.
- Do not change approval, sandbox, model-provider, or telemetry settings unless the user explicitly requests that behavior.
- Do not use `git reset`, `git checkout`, or broad deletion to clean up the worktree.
- For a request to record or activate an existing skill, use the project's `skm` commands and preserve the distinction between linking and copying.
