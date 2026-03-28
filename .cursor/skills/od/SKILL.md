---
name: od
description: >-
  OmniDev AI-driven development workflow. Use when the user types /od to start
  a requirement, /od help for commands, /od onboard for legacy project scanning,
  /od report for status reports, /od review for code review, /od qa for testing,
  or /od change for mid-stream requirement changes.
---

# OmniDev Workflow Skill

When the user triggers `/od`, strictly follow the OmniDev workflow defined in the project's `.cursor/rules/01-omnidev-workflow.mdc`.

## Quick Reference

| Command | Action |
|---------|--------|
| `/od [requirement]` | Standard workflow: assess complexity -> blueprint -> plan -> develop |
| `/od --fast [requirement]` | Skip blueprint/plan, develop directly (hotfixes) |
| `/od --skip <phases> [req]` | Skip phases (`blueprint`,`plan`,`dev`,`test`), also supports natural language (e.g., "跳过测试") |
| `/od --plan-only [requirement]` | Output blueprint and plan only, no coding |
| `/od help` | Show all OmniDev commands |
| `/od onboard` | Scan legacy project, generate context document |
| `/od report` | Generate enterprise-grade weekly status report |
| `/od review` | Code review only (no modifications) |
| `/od qa` | Write and execute test cases |
| `/od change [new requirement]` | Handle mid-stream requirement changes |
| `/od learn` | Self-learning from recent errors |
| `/od update` | Fetch latest rules from remote repo, merge & audit |
| `/od compress` | Manually trigger context memory compression |

## Key Behaviors

- **Natural language support**: After the `/od` prefix, commands can be expressed in natural language (e.g., `/od 帮我review代码` = `/od review`). The `/od` prefix is required — arbitrary conversation does NOT trigger OmniDev.
- **Phase navigation**: After each phase completes, output a progress summary showing completed/current/upcoming phases, and ask before proceeding.
- **Mid-phase adjustments**: At every checkpoint, the user can adjust current output, skip future phases, insert ad-hoc steps, or go back to a previous phase.

## Workflow

1. **Read the full OmniDev rules** from `.cursor/rules/01-omnidev-workflow.mdc` before proceeding.
2. **Parse intent** from slash command OR natural language, identify which flow to execute.
3. **Assess complexity** (S/M/L/XL) using the 5-dimension scoring rubric.
4. **Follow the phased workflow** with checkpoints — never write business code before requirements are confirmed.
5. **Store all state documents** in `docs/omnidev-state/`.
6. **Reply in the user's language** (Chinese if they write in Chinese).

## Related Rules

All OmniDev rules are in `.cursor/rules/`:
- `01-omnidev-workflow.mdc` — Core workflow phases
- `02-omnidev-state-sync.mdc` — State persistence and session recovery
- `03-omnidev-test-deploy.mdc` — Testing and deployment
- `04-omnidev-skills-mcp.mdc` — Skills and MCP integration
- `05-omnidev-context-management.mdc` — Context pruning
