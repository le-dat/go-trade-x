# Command: checkpoint

## Description

Unified command to update project documentation (changelog, status, plan) and summarize the current work session. Auto-generates docs/ folder if it doesn't exist.

## When asked to run this command, Claude must:

### Step 0 — Ensure docs/ folder exists

Check if `docs/` folder exists. If not, create it with template files:

```bash
mkdir -p docs
```

Create the following template files if missing:

**`docs/changelog.md`**:
```markdown
# Changelog

> All notable changes to this project are recorded here.
> Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## Unreleased

### Added
-

### Changed
-

### Fixed
-
```

**`docs/project-plan.md`**:
```markdown
# Project Plan

> Generated from project specification. Update via `/checkpoint` after each milestone.

## Milestones

### Milestone 1 — MVP
- [ ] Feature 1
- [ ] Feature 2

### Milestone 2 — Beta
- [ ] Feature 3
```

**`docs/project-status.md`**:
```markdown
# Project Status

> Last updated: YYYY-MM-DD

## Current Phase
**Phase 1: Planning**

## Overall Progress
**0%** ░░░░░░░░░░░░░░░░░░░░

## Current Status
- ✅ Completed: (none yet)
- 🔄 In Progress: (none yet)
- 📋 Next: Define project scope in CLAUDE.md

## Session History

### YYYY-MM-DD — Session 1
- Initialized project

## Next Session — Start Here
1. Read CLAUDE.md to understand project
2. Run `/generate-plan` to create roadmap
```

**`docs/spec-doc.md`**:
```markdown
# Project Specification

> Define your project vision, goals, and requirements here.

## Overview
**Project Name:** {{PROJECT_NAME}}
**Type:** {{PROJECT_TYPE}}

## Goals
1. Goal 1
2. Goal 2

## Requirements
### Must Have
-

### Nice to Have
-
```

**`docs/architecture.md`**:
```markdown
# Architecture

> System architecture and technical decisions.

## Tech Stack
-

## Directory Structure
-

## Key Components
-
```

### Step 1 — Analyze recent work

- Read `git log --oneline -n 10` or check current memory for changes made in the session.
- Identify all completed items, even if they are sub-tasks of a larger step.
- Identify:
  - ✅ **Completed items** (including "Grouped Tasks" — multiple related sub-tasks).
  - 🔄 **In-progress work**.
  - 🐛 **New bugs** or technical debt.
  - 💡 **Architectural decisions** made.

### Step 2 — Update Documentation (Unified & Grouped)

1. **`docs/changelog.md`**:
   - Add new entry at the top for current session.
   - **Group related tasks:** Instead of many redundant lines, group sub-tasks under a logical feature header.
   - Format: Date, Title, Added/Changed/Fixed lists.

2. **`docs/project-plan.md`**:
   - Find completed steps and mark them with `✅`.
   - **Handling Hierarchical Tasks (Grouped Tasks):** If a Step has an internal list (e.g., Step 8: Build Modules), mark only the specific sub-items that are done (e.g., `1. Auth ✅`).
   - If *all* sub-items within a step are complete, mark the main Step as `✅`.
   - Update status of in-progress or blocked steps.

3. **`docs/project-status.md`**:
   - Update **Current Phase** and **Overall progress %**.
   - Update **Last Session** with the date and **grouped** tasks summaries.
   - Update **Next Session — Start Here** with clear goals and starting prompts.
   - Refresh **Milestone Progress** tables.

4. **`CLAUDE.md` / `docs/architecture.md`** (Optional):
   - Update only if significant patterns or architectural changes were made.

### Step 3 — Maintenance (Optional)

- Suggest running `/commit` if there are untracked/modified files that should be saved.
- Prompt for GitHub issue creation if new bugs or ideas were found.

### Step 4 — Summary

- Summarize what was updated in the documentation.
- Provide a starting prompt or recommended command for the next session.
