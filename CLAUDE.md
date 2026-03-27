# armory — Skill Control Plane for AI Agents

## Project
Go CLI/TUI that manages skill presets for AI agents. Equip terminals with role-based skill loadouts.

## Commands
```bash
go build -o armory .                    # Build
go test ./...                           # Run all tests
go test ./internal/scanner/...          # Test scanner only
go test ./internal/config/...           # Test config only
go test ./internal/state/...            # Test state only
go test -v ./... -run TestName          # Run specific test
go vet ./...                            # Lint
```

## Architecture
```
main.go                    # Entry point
cmd/                       # cobra command definitions
  root.go                  # Root command + version
  scan.go                  # armory scan
  role.go                  # armory role {list,create,edit,show}
  equip.go                 # armory equip <role>
  unequip.go               # armory unequip
  board.go                 # armory board
internal/
  scanner/scanner.go       # Skill indexing, frontmatter parsing, cache
  config/config.go         # armory.yaml parsing, global + project merge
  state/state.go           # ~/.armory/state.json CRUD, symlink verify
  tui/picker.go            # bubbletea fuzzy skill picker
  tui/board.go             # bubbletea board dashboard
```

## Key Decisions (from eng review)
- **CLI framework:** cobra (subcommands, flags, completion)
- **Categories:** Keyword inference from skill names (no category field in real SKILL.md files)
- **Duplicates:** First-match-wins by skill_paths order in config
- **Board:** Verifies symlinks on refresh (stale/broken detection)
- **Role edit:** Re-opens picker with pre-checked selections
- **Config merge:** Project-level fully replaces global roles (no deep merge)
- **Overlap guard:** Equip checks target dir doesn't overlap with source skill_paths

## Conventions
- Table-driven tests with testify/assert
- Use t.TempDir() for filesystem tests
- All public functions have doc comments
- Error messages should be user-friendly (no stack traces)
- Use lipgloss for styled terminal output

## Git
- Email: maketroli@gmail.com
- Conventional commits: feat, fix, test, chore, docs
