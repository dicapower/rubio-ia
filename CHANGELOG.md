# Changelog

## v0.1.0

First release.

- `rubio setup` writes a short working agreement (about 470 tokens) into Claude Code (`~/.claude/CLAUDE.md`) and Cursor (`~/.cursor/rules/rubio-ia.mdc`), inside a marked block that never touches your own text.
- Language rule: `auto` (answer in the language you write in), `es-ES` (Spanish from Spain) or `en`.
- `rubio-plan` skill: plan, approve, build, verify for large or risky changes, with an independent review for high-risk ones.
- `rubio remove`, `rubio restore` and `rubio doctor`. Every change is backed up first; the ten latest backups are kept.
- `rubio migrate gentle-ai`: removes Gentle-AI's blocks, output style and rule file. It prints a plan and changes nothing until you pass `--yes`.
- No network access, no telemetry, no cgo. Installs with `install.sh`, Homebrew or Scoop.
