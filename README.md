# armory

**Skill control plane for AI agents.** Equip your terminals with team-based skill presets — zero manual symlinking.

Just run `armory`:

```
$ armory

  ARMORY v0.2.0

  > Equip a terminal
    Browse & install skills
    Manage teams
    Board
    Settings

  3 teams · 143 skills · 2 equipped
```

First time? armory walks you through setup:

```
  Welcome to armory

  Select your teams (space to toggle, enter to confirm)

  ✓ Dev          Software development and code quality
  ✓ Marketing    Content creation, SEO, and brand work
    QA           Quality assurance and testing
  > Design       UI/UX design and visual quality
    ...

  Setting up your teams...

  Dev:
    Found locally: best-practices, performance, systematic-debugging (3/8)
    Available from skills.sh: ship, review (2 more)

  Marketing:
    Found locally: copywriting, seo (2/14)
    Available from skills.sh: content-strategy, email-sequence + 10 more

  Install 12 skills from skills.sh? [Y/n]
```

Then equip any terminal in one command:

```
$ armory equip marketing
  Searching locally...
    Found: copywriting, marketing-psychology, seo (3/14)
  Searching skills.sh...
    Found: content-strategy, email-sequence + 9 more
  Installing 11 skills...
    + content-strategy → ~/.agents/skills/content-strategy
    + email-sequence → ~/.agents/skills/email-sequence
    ...
  Creating symlinks...
    ✓ copywriting
    ✓ marketing-psychology
    ✓ seo
    ✓ content-strategy
    ...

  Equipped 'marketing' team: 14 skills loaded
```

**Every terminal gets an identity.**

## Why

The AI agent tooling space has session managers (Claude Squad, Agent Deck, Conduit). What's missing is the **skill layer** — a tool that maps teams to skill presets and equips agents automatically.

armory connects you to the [skills.sh](https://skills.sh) ecosystem — 90K+ installs, curated collections from Anthropic, Vercel, and the community. It discovers what you have locally, finds what you're missing, and installs it.

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

```bash
# Interactive mode — just run it
armory

# Or use subcommands directly
armory scan                     # List all available skills
armory team create marketing    # Create a team interactively
armory equip marketing          # Equip current directory
armory board                    # See all equipped directories
```

## Commands

```
armory                          Launch interactive shell (setup wizard on first run)
armory scan                     List all available skills
armory team list                List configured teams
armory team create <name>       Create a team (interactive picker)
armory team edit <name>         Edit a team's skills
armory team show <name>         Show team details
armory equip <team>             Equip current directory with a team
armory equip <team> --dir path  Equip a specific directory
armory equip <team> --merge     Keep existing skills, only add new ones
armory unequip                  Remove armory-managed symlinks
armory board                    Dashboard of all equipped directories
```

## How It Works

```
skills.sh registry          ~/.claude/skills/   ~/.agents/skills/
       │                         │                      │
       │  (install missing)      └──────┬───────────────┘
       │                               ▼
       └──────────────────►  armory scan (indexes all skills)
                                       │
                                       ▼
                              armory.yaml (team definitions)
                                       │
                                       ▼
                              armory equip marketing
                                       │
                              ┌────────┴────────┐
                              ▼                 ▼
                     .claude/skills/      ~/.armory/
                     (symlinks)           state.json
                              │
                              ▼
                        armory board
                        (dashboard)
```

**Key concepts:**

- **Teams** are named presets that map to a list of skills (e.g., "marketing" → copywriting, seo, content-strategy...)
- **Skill paths** (`~/.claude/skills/`, `~/.agents/skills/`) are source directories where skills live
- **Equip** creates symlinks from a project's `.claude/skills/` to the source skill directories
- **Registry integration** — when skills are missing locally, armory searches skills.sh and offers to install them
- **State tracking** — armory tracks which symlinks it created, enabling safe unequip
- **Board verification** — the dashboard verifies symlinks on refresh, showing "stale" or "broken" status

## Built-in Team Presets

armory ships with 10 team presets ready to use:

| Team | Skills | Registry |
|------|--------|----------|
| Dev | best-practices, performance, tdd, debugging, ... | — |
| Marketing | copywriting, seo, content-strategy, email-sequence, ... | coreyhaines31/marketingskills |
| QA | qa, browse, benchmark, web-quality-audit | — |
| Design | frontend-design, accessibility, core-web-vitals, ... | — |
| Security | codeql, semgrep, yara-rule-authoring, ... | — |
| Planning | office-hours, plan-eng-review, brainstorming | — |
| DevOps | land-and-deploy, setup-deploy, canary | — |
| Frontend | gsap, react-best-practices, remotion, tailwind | — |
| Backend | best-practices, performance, debugging, tdd | — |
| Data | analytics-tracking, ab-test-setup | — |

## Configuration

```yaml
# ~/.armory/armory.yaml
version: 2

skill_paths:
  - ~/.claude/skills
  - ~/.agents/skills

teams:
  marketing:
    description: "Content creation, SEO, and brand work"
    skills:
      - copywriting
      - marketing-psychology
      - seo
      - content-strategy
    missing_action: prompt
```

| Path | Purpose |
|------|---------|
| `~/.armory/armory.yaml` | Global config (teams, skill paths) |
| `./armory.yaml` | Project-level config (overrides global) |
| `~/.armory/state.json` | Tracks equipped directories |
| `~/.armory/cache/skills.json` | Cached skill index |

## Works With

- **Claude Code** (native — skills load from `.claude/skills/`)
- **Claude Squad**, **Agent Deck**, **Conduit** (equip before starting)
- **Any terminal emulator** (Ghostty, iTerm2, Kitty, xterm)
- **skills.sh** (registry for discovering and installing skills)

## Building from Source

```bash
git clone https://github.com/m-ret/armory.git
cd armory
go build -o armory .
go test ./...
```

## License

MIT
