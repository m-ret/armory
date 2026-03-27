# armory — Skill Control Plane for AI Agents

## Project
Go CLI/TUI that manages skill presets for AI agents. Equip terminals with team-based skill loadouts. Interactive shell with setup wizard and skills.sh registry integration.

## Commands
```bash
go build -o armory .                    # Build
go test ./...                           # Run all tests
go test ./internal/scanner/...          # Test scanner only
go test ./internal/config/...           # Test config only
go test ./internal/state/...            # Test state only
go test ./internal/registry/...         # Test registry only
go test ./internal/teams/...            # Test team presets
go test -v ./... -run TestName          # Run specific test
go vet ./...                            # Lint
```

## Architecture
```
main.go                    # Entry point
cmd/                       # cobra command definitions
  root.go                  # Root command + interactive shell launch
  scan.go                  # armory scan
  team.go                  # armory team {list,create,edit,show}
  equip.go                 # armory equip <team> (with registry integration)
  unequip.go               # armory unequip
  board.go                 # armory board
internal/
  scanner/scanner.go       # Skill indexing, frontmatter parsing, cache
  config/config.go         # armory.yaml parsing, global + project merge
  state/state.go           # ~/.armory/state.json CRUD, symlink verify
  teams/presets.go         # Hardcoded team presets with skills.sh mappings
  registry/registry.go     # skills.sh search + install (npx/git hybrid)
  tui/shell.go             # Interactive shell (main menu)
  tui/wizard.go            # First-run setup wizard
  tui/picker.go            # bubbletea fuzzy skill picker
  tui/board.go             # bubbletea board dashboard
```

## Key Decisions
- **Interactive-first:** `armory` bare launches TUI shell; subcommands for power users
- **Teams (not roles):** User-facing terminology is "teams"
- **CLI framework:** cobra (subcommands, flags, completion)
- **Categories:** Keyword inference from skill names
- **Duplicates:** First-match-wins by skill_paths order in config
- **Board:** Verifies symlinks on refresh (stale/broken detection)
- **Team edit:** Re-opens picker with pre-checked selections
- **Config merge:** Project-level fully replaces global teams (no deep merge)
- **Registry:** Hybrid install — npx skills if available, git clone if not
- **Presets:** 10 team presets embedded in binary with skills.sh collection mappings

## Conventions
- Table-driven tests with testify/assert
- Use t.TempDir() for filesystem tests
- All public functions have doc comments
- Error messages should be user-friendly (no stack traces)
- Use lipgloss for styled terminal output

## Git
- Email: maketroli@gmail.com
- Remote: git@github-personal:m-ret/armory.git (MUST use github-personal SSH alias)
- Conventional commits: feat, fix, test, chore, docs
