# armory

**Skill control plane for AI agents.** Equip your terminals with role-based skill presets — zero manual symlinking.

You run 3 agents at once: one for marketing, one for development, one for planning. Each needs different skills loaded. Instead of manually choosing and symlinking skills per terminal, you run one command:

```
$ armory equip marketing
  + copywriting
  + marketing-psychology
  + seo
  + browse
  + content-strategy

Equipped 'marketing' role: 5 skills loaded
```

Open another terminal:

```
$ cd ~/Work/api && armory equip dev
  + best-practices
  + performance
  + test-driven-development
  + systematic-debugging
  + verification-before-completion

Equipped 'dev' role: 5 skills loaded
```

See everything at a glance:

```
$ armory board
┌─────────────────────────────────────────────────────────┐
│  ARMORY BOARD                              2 equipped   │
├──────────────┬───────────┬────────┬──────────┬──────────┤
│ Directory    │ Role      │ Skills │ Status   │ Since    │
├──────────────┼───────────┼────────┼──────────┼──────────┤
│ ~/Work/myapp │ marketing │ 5/5    │ equipped │ 3m ago   │
│ ~/Work/api   │ dev       │ 5/7    │ equipped │ 1m ago   │
└──────────────┴───────────┴────────┴──────────┴──────────┘
```

**Every terminal gets an identity.**

## Why

The AI agent tooling space has session managers (Claude Squad, Agent Deck, Conduit). What's missing is the **skill layer** — a tool that maps roles to skill presets and equips agents automatically.

armory is that tool. It's an armory in the gaming sense: store your loadouts by role, equip before the mission.

## Install

```bash
# Go install (requires Go 1.22+)
go install github.com/m-ret/armory@latest

# Homebrew (macOS/Linux)
brew install m-ret/tap/armory

# Or download a binary from GitHub Releases
# https://github.com/m-ret/armory/releases
```

## Quick Start

### 1. See what skills you have

```bash
armory scan
```

This scans `~/.claude/skills/` and `~/.agents/skills/` and shows a table of all available skills with their inferred categories.

### 2. Create a role

```bash
armory role create marketing
```

This opens an **interactive fuzzy picker** — type to filter, space to toggle, enter to confirm. Skills are grouped by category.

Or define roles directly in `~/.armory/armory.yaml`:

```yaml
version: 1

skill_paths:
  - ~/.claude/skills
  - ~/.agents/skills

roles:
  marketing:
    description: "Content creation, SEO, and brand work"
    skills:
      - copywriting
      - marketing-psychology
      - seo
      - browse
      - content-strategy
    missing_action: prompt   # prompt | skip | error

  dev:
    description: "Software development and code quality"
    skills:
      - best-practices
      - performance
      - test-driven-development
      - systematic-debugging
      - verification-before-completion
    missing_action: prompt

  planning:
    description: "Architecture and project planning"
    skills:
      - office-hours
      - plan-eng-review
      - plan-ceo-review
      - plan-design-review
      - brainstorming
    missing_action: prompt
```

### 3. Equip a terminal

```bash
cd ~/Work/myapp
armory equip marketing
```

This creates symlinks in `.claude/skills/` pointing to the source skill directories. Start Claude Code and the skills are loaded automatically.

### 4. See the dashboard

```bash
armory board
```

A TUI dashboard showing all equipped directories, their roles, skill counts, and status. Press `r` to refresh, `q` to quit.

## Commands

```
armory scan                     List all available skills
armory role list                List configured roles
armory role create <name>       Create a role (interactive picker)
armory role edit <name>         Edit a role's skills (picker with pre-selections)
armory role show <name>         Show role details
armory equip <role>             Equip current directory with a role
armory equip <role> --dir path  Equip a specific directory
armory equip <role> --merge     Keep existing skills, only add new ones
armory unequip                  Remove armory-managed symlinks
armory unequip --dir path       Unequip a specific directory
armory board                    Dashboard of all equipped directories
```

## How It Works

```
~/.claude/skills/     ~/.agents/skills/      <- skill source directories
       │                      │
       └──────┬───────────────┘
              ▼
        armory scan                          <- indexes skills, caches results
              │
              ▼
       armory.yaml                           <- role definitions (skill presets)
              │
              ▼
       armory equip marketing                <- creates symlinks
              │
              ▼
    ~/Work/myapp/.claude/skills/             <- symlinks to source skills
        copywriting -> ~/.agents/skills/copywriting
        seo -> ~/.agents/skills/seo
        ...
              │
              ▼
       ~/.armory/state.json                  <- tracks what armory created
              │
              ▼
       armory board                          <- reads state, verifies symlinks
```

**Key concepts:**

- **Skill paths** (`skill_paths` in config) are source directories where skills live. armory reads from them but never writes to them.
- **Equip target** is always a project's `.claude/skills/` directory (the current working directory by default).
- **State tracking** — armory tracks which symlinks it created in `~/.armory/state.json`. This enables safe `unequip` (only removes what armory put there, never your manual symlinks).
- **Board verification** — the dashboard verifies symlinks still exist on every refresh. If someone deletes a symlink manually, the board shows "stale" status.

## Conflict Handling

**Replace mode** (default): When you equip a directory that's already equipped, armory removes the old managed symlinks and creates new ones. Manual symlinks are never touched.

```bash
armory equip dev        # Equips dev role
armory equip marketing  # Warns "Replacing 'dev' with 'marketing'", then replaces
```

**Merge mode**: Keep existing skills and only add missing ones.

```bash
armory equip dev
armory equip marketing --merge  # Adds marketing skills alongside dev skills
```

## Missing Skills

When a role references a skill that doesn't exist in any skill path:

```
⚠ Missing skills for 'marketing' role:
  - social-content (not found in any skill path)

Options:
  [s] Skip missing skills and continue
  [a] Abort
```

The behavior is controlled by `missing_action` in the role config:
- `prompt` — ask the user (falls back to `skip` in non-interactive mode)
- `skip` — silently skip missing skills
- `error` — fail if any skills are missing

## Project-Level Config

Place an `armory.yaml` in your project root to override global roles for that project:

```yaml
version: 1
roles:
  dev:
    description: "Dev with project-specific skills"
    skills:
      - best-practices
      - my-custom-project-skill
    missing_action: skip
```

Project roles **fully replace** global roles with the same name. To extend a global role, copy it and add your skills.

## Skill Format

armory reads skills from directories containing a `SKILL.md` file with YAML frontmatter:

```yaml
---
name: copywriting
description: Professional copywriting and content creation
---
```

The `name` and `description` fields are used for the scan table and interactive picker. Categories are inferred automatically from skill names — no `category` field needed.

## Configuration

| Path | Purpose |
|------|---------|
| `~/.armory/armory.yaml` | Global config (roles, skill paths) |
| `./armory.yaml` | Project-level config (overrides global) |
| `~/.armory/state.json` | Tracks equipped directories |
| `~/.armory/cache/skills.json` | Cached skill index |

## Works With

armory is a configuration tool — it sets up skill symlinks and gets out of the way. It works with:

- **Claude Code** (native — skills load from `.claude/skills/`)
- **Claude Squad** (equip before starting a squad session)
- **Any terminal emulator** (Ghostty, iTerm2, Kitty, xterm, etc.)
- **Any session manager** (tmux, zellij, screen, etc.)

## Building from Source

```bash
git clone https://github.com/m-ret/armory.git
cd armory
go build -o armory .
go test ./...
```

## License

MIT
