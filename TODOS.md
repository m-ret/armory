# TODOS — armory

## v2: MCP Protocol Support for Skills
**What:** Add MCP (Model Context Protocol) as a skill/tool discovery protocol alongside filesystem scanning.
**Why:** The AI tooling ecosystem (Cline, Claude Code, etc.) is converging on MCP for tool definitions. Supporting it makes armory compatible with the broader ecosystem — users get access to any MCP-compatible tool server, not just local skill directories.
**Pros:** Ecosystem compatibility, more skill sources, future-proof.
**Cons:** Protocol overhead, added complexity, need to handle remote MCP servers.
**Context:** Cline (github.com/cline/cline) has mature MCP integration with marketplace discovery and remote server configs. v1 uses filesystem-based skill scanning (reads SKILL.md frontmatter from ~/.claude/skills/ and ~/.agents/skills/). MCP would be additive, not replacing filesystem scanning.
**Depends on:** v1 filesystem skill scanning working and validated.
**Added:** 2026-03-26 via /plan-eng-review

## v2: Role Inheritance
**What:** Allow roles to extend other roles. e.g., `fullstack` extends `dev` + adds `frontend-design`, `accessibility`.
**Why:** Users with overlapping roles (dev + design, planning + dev) currently need to duplicate skill lists. Inheritance lets them compose roles from smaller building blocks.
**Pros:** DRY role definitions, easier to maintain, enables composable skill presets.
**Cons:** Adds complexity to config parsing (cycle detection, override semantics), harder to debug "where did this skill come from?"
**Context:** Deferred from v1 design doc open questions. v1 uses flat skill lists per role. Inheritance would add an `extends` field to role definitions.
**Depends on:** v1 role management working and validated.
**Added:** 2026-03-26 via /plan-eng-review

## v2: Hash-Based Cache Invalidation
**What:** Replace directory mtime-based cache invalidation with content hashing of SKILL.md files.
**Why:** Directory mtime only detects file adds/removes, not edits to SKILL.md content (name, description changes). Hash-based invalidation catches all changes.
**Pros:** More accurate cache invalidation, catches metadata edits.
**Cons:** Slightly slower (must read + hash each SKILL.md vs single stat call per dir). Negligible for ~150 skills.
**Context:** Identified by Codex outside voice during eng review. v1 uses mtime which is sufficient for the common case (adding/removing skill directories). Content edits are rare.
**Depends on:** v1 scanner + cache working.
**Added:** 2026-03-26 via /plan-eng-review (Codex outside voice)
