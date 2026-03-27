# TODOS — fleetdeck

## v2: MCP Protocol Support for Skills
**What:** Add MCP (Model Context Protocol) as a skill/tool discovery protocol alongside filesystem scanning.
**Why:** The AI tooling ecosystem (Cline, Claude Code, etc.) is converging on MCP for tool definitions. Supporting it makes fleetdeck compatible with the broader ecosystem — users get access to any MCP-compatible tool server, not just local skill directories.
**Pros:** Ecosystem compatibility, more skill sources, future-proof.
**Cons:** Protocol overhead, added complexity, need to handle remote MCP servers.
**Context:** Cline (github.com/cline/cline) has mature MCP integration with marketplace discovery and remote server configs. v1 uses filesystem-based skill scanning (reads SKILL.md frontmatter from ~/.claude/skills/ and ~/.agents/skills/). MCP would be additive, not replacing filesystem scanning.
**Depends on:** v1 filesystem skill scanning working and validated.
**Added:** 2026-03-26 via /plan-eng-review

## v2: Agent Approval Model for Multi-Agent Operation
**What:** Per-agent trust levels: auto-approve (full trust), notify-only (see actions, no gate), require-approval (current default).
**Why:** With 8 agents running simultaneously, you can't approve every action in every pane. Cline's single-agent model (approve every action) doesn't scale. Users need to set trust levels per agent so autonomous operation is possible while maintaining control over sensitive agents.
**Pros:** Enables true autonomous multi-agent workflows, scales human attention.
**Cons:** Security implications (auto-approve means trusting the agent fully), needs careful UX design for trust level configuration.
**Context:** v1 ships with Claude Code's native approval model (whatever the user has configured in their Claude Code settings). This TODO is about fleetdeck adding its own approval layer on top. Related: Autonomous Mode (v2) from the design doc — agent-to-agent work passing also needs trust boundaries.
**Depends on:** v1 agent lifecycle working, understanding Claude Code's approval/permission model.
**Added:** 2026-03-26 via /plan-eng-review

## v2: Agent Checkpoint/Rollback
**What:** Snapshot agent workspace state at key points so users can rewind an agent to a known-good state.
**Why:** With concurrent agents modifying code, one agent can trash a codebase while you're watching another. Per-agent checkpoints provide a safety net — rewind to before the bad change.
**Pros:** Safety net for concurrent operation, user confidence, enables experimentation.
**Cons:** Storage overhead (git snapshots? file copies?), complexity of state management, need to handle in-flight operations during rollback.
**Context:** Cline implements this as workspace snapshots at each step. For fleetdeck, the natural approach would be git-based: auto-commit or stash before agent actions, allow reverting to any checkpoint. Depends on whether agents work in separate git branches/worktrees (which would make this simpler) or shared workspaces.
**Depends on:** v1 agent lifecycle, understanding Claude Code's own checkpoint mechanism (if any), deciding whether agents use separate worktrees.
**Added:** 2026-03-26 via /plan-eng-review
