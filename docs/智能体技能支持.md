# Agent Skill Support

SKM manages user-level Agent Skills as one directory per Skill. Every supported
target receives the complete Skill directory, whose entry point is `SKILL.md`.
The file uses YAML frontmatter with at least `name` and `description`, followed
by Markdown instructions. Supporting files remain beside it in the same Skill
directory.

| Agent | SKM ID | User-level Skill directory | Status |
| --- | --- | --- | --- |
| Claude Code | `claude` | `~/.claude/skills/<skill-name>/` | Supported |
| Codex | `codex` | `~/.codex/skills/<skill-name>/` | Supported |
| Cursor | `cursor` | `~/.cursor/skills/<skill-name>/` | Supported |
| GitHub Copilot | `copilot` | `~/.copilot/skills/<skill-name>/` | Supported |
| Gemini CLI | `gemini` | `~/.gemini/skills/<skill-name>/` | Supported |
| Windsurf | `windsurf` | `~/.codeium/windsurf/skills/<skill-name>/` | Supported |
| Kiro | `kiro` | `~/.kiro/skills/<skill-name>/` | Supported |
| Cline | `cline` | `~/.agents/skills/<skill-name>/` | Supported |
| OpenCode | `opencode` | `~/.config/opencode/skills/<skill-name>/` | Supported |
| Trae | `trae` | `~/.trae/skills/<skill-name>/` | Supported |
| Hermes | `hermes` | `~/.hermes/skills/<skill-name>/` | Supported |
| OpenClaw | `openclaw` | `~/.openclaw/skills/<skill-name>/` | Supported |

Claude Code and Codex are fixed in Agent management. Other supported Agents
must be added there before a Skill can be enabled for them. An Agent cannot be
removed from management while it still has an enabled user-level Skill.

The directory matrix follows the supported-Agent registry maintained by the
[Vercel Skills CLI](https://github.com/vercel-labs/skills/blob/main/README.md).
Gemini CLI also documents the shared `SKILL.md` layout in its
[Agent Skills codelab](https://codelabs.developers.google.com/gemini-cli/how-to-create-agent-skills-for-gemini-cli).
Custom Agents can be added with a unique ID, display name, optional local icon,
and a user-level Skill root under `~/`. SKM stores these definitions in
`config.yaml`; they use the same `<skill-name>/SKILL.md` layout.
