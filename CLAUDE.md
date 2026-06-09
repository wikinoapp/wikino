<!-- last_synced: 2026-06-09 -->

# Wikino Development Guide

> English | [日本語](./CLAUDE.ja.md)

This file provides guidance for Claude Code when working in this repository.

## Overview

Wikino is a wiki application.
Users create pages in places called "spaces" and connect pages with links.

## Project Structure

This repository manages two subprojects—the Go version and the Rails version—as a monorepo:

```
/workspace/
├── go/                  # Go version implementation (features being migrated gradually)
├── rails/               # Rails version implementation (existing production system)
├── caddy/               # Reverse proxy configuration
├── docs/                # Wikino-specific documentation (ADR, work plans, etc.)
├── .github/             # Shared CI/CD configuration
├── Dockerfile.dev       # Dockerfile for the integrated development container
├── docker-compose.yml   # Docker Compose configuration
├── mise.toml            # Development tool version management (Go, Ruby)
└── CLAUDE.md            # This file (project-wide guide)
```

## Rails to Go Migration

A project to gradually reimplement the existing Rails Wikino in Go is currently underway.

### Migration Strategy

- **Use the existing DB as-is**: Share the PostgreSQL database managed on the Rails side
- **Gradual migration**: Rails and Go share the same DB and session store, and features are migrated incrementally
- **Data migrations run on the Go side**: Use the migration mechanism provided on the Go side (dbmate)
- **Continued use of shared infrastructure**: Shared infrastructure such as PostgreSQL continues to be used after the Go version takes over
- **Do not modify the Rails source code**: When a feature needs to be added or changed, migrate it to Go first rather than touching the Rails side
  - The following cases fall outside this principle, and a minimal-diff fix on the Rails side is acceptable:
    - Minimal maintenance changes required to follow up on a dependency's security fix (e.g., adapting to breaking changes from a gem major upgrade)
    - When deleting Rails-side processing that has become unused after migrating the feature to Go

When implementing the Go version, refer to the Rails code to understand the existing specifications.

## Feature Flag-Based Development

Wikino uses **feature flags** rather than feature branches to control feature visibility. Pre-release features are developed with the flag off, and the flag is flipped to release them once they are ready for production.

## Development Workflow

### Implementation Guidelines

**Consistency with existing code**:

Before implementing, check whether similar processing already exists in the codebase.
If similar processing exists, follow that pattern to maintain consistency across the codebase.

### Post-Implementation Checks

Before announcing that work is complete, always confirm the following:

- Code formatting
- Lint
- Tests

The commands to run are managed in the `Makefile`.
See [Makefile](./Makefile), [go/Makefile](./go/Makefile), and [rails/Makefile](./rails/Makefile).

## Documentation

The current state of the system (validation, permissions, structure) is owned by the code and tests. Documentation focuses on the _why_ behind design decisions, recorded as **ADR (Architecture Decision Record)**, and on in-progress changes (work plans, reviews).

To understand why something is built the way it is, read the ADRs first; to understand how it currently behaves, read the code and tests.

- [docs/README.md](./docs/README.md) - Documentation guide. ADRs live in `docs/private/adr/`

## Language and Writing Conventions

- **Canonical version is English; authoring workflow is Japanese-first**: The English version is the official authoritative source. Author by writing Japanese first, then translate to English (Claude Code assists). After translation, also review the English version to catch meaning drift and unnatural wording. When a discrepancy arises, the English version takes precedence
- **Code comments**: English block → blank line → Japanese block prefixed with `[Ja]`. Short comments can be one-line pairs like `# Returns ... / [Ja] ... を返す`
- **Markdown documents**: Maintain `xxx.md` (English, canonical) and `xxx.ja.md` (Japanese translation) in parallel. Both files carry a `last_synced: YYYY-MM-DD` field in the YAML frontmatter; keep the dates aligned
- **Commit messages**: English title + English body + blank line + Japanese body prefixed with `[Ja]`. Do not preserve a Japanese title (prioritize English scannability of `git log --oneline`)
- **Identifiers**: Type, function, and variable names are English only
- **Update both sides in the same commit**: Prevents translation drift
- **Existing code**: Apply this rule to new writing. Migrate existing monolingual code to bilingual when editing it (no bulk migration required)

## Coding Conventions

- Queries against tables that have a `space_id` column must always include `space_id` in the WHERE clause

```sql
-- ✅ Good: Include space_id directly in the condition when the table has it
UPDATE pages SET title = $2 WHERE id = $1 AND space_id = $3;
DELETE FROM draft_pages WHERE id = $1 AND space_id = $2;

-- ✅ Good: Verify via JOIN when the table doesn't have space_id
SELECT par.* FROM page_attachment_references par
INNER JOIN pages p ON par.page_id = p.id
WHERE par.page_id = $1 AND p.space_id = $2;

-- ❌ Bad: Operating only by ID without space_id
UPDATE pages SET title = $2 WHERE id = $1;
DELETE FROM draft_pages WHERE id = $1;
SELECT * FROM page_attachment_references WHERE page_id = $1;
```

- For environment variables defined by Wikino, always use the `WIKINO_` prefix, except for those required by external libraries
