# Rebaser Agent Plan

Adopt the persona from hack/agent-rebaser.md

## Your Mission

You are tasked with rebasing the current `rb` branch commits onto `main`, squashing them appropriately, and creating a rich commit message following the PR template format.

## Current Situation

- **Current Branch**: `rb` 
- **Target Branch**: `main`
- **Recent Commits on rb**:
  - 2dfe1a7 wip
  - 36013e3 Improve worker isolation and Claude integration  
  - 6beb76d cleanup plans
  - 31a9859 Add KUBECONFIG isolation test plan
  - 3d4b67b Add comprehensive progress report

## What You Need to Do

1. **Read the complete diff** from main to rb branch (minimum 1500 lines)
2. **Understand the feature** - This appears to be work on agent personas and worker isolation
3. **Create backup branch** before starting rebase
4. **Rebase onto main** and squash related commits
5. **Write rich commit message** using the PR template format from `../humanlayer/humanlayer/.github/PULL_REQUEST_TEMPLATE.md`

## Key Focus Areas

Based on the commit messages, this work involves:
- Agent persona system (hack/agent-*.md files)
- Worker isolation and tmux integration  
- KUBECONFIG isolation for testing
- Claude code integration improvements
- Progress reporting enhancements

## Expected Outcome

A single, well-crafted commit on main that:
- Combines all the rb branch work logically
- Has a comprehensive commit message following the PR template
- Explains the problems solved and user-facing changes
- Includes technical implementation details
- Provides verification steps

## Commit Message Structure

Use this format (from PR template):
```
feat(personas): implement agent persona system with worker isolation

## What problem(s) was I solving?

[Describe the problems this work addresses]

## What user-facing changes did I ship?

[List the user-visible improvements]

## How I implemented it

[Technical implementation details]

## How to verify it

[Steps to test the changes]

## Description for the changelog

[Concise summary for users]
```

## Critical Requirements

- **MUST** read at least 1500 lines of diff to understand complete context
- **MUST** create backup branch before rebasing
- **MUST** verify tests pass after rebase
- **MUST** commit every 5-10 minutes during work
- **MUST** follow the persona guidelines exactly

## Success Criteria

- [ ] Clean rebase onto main completed
- [ ] All related commits properly squashed
- [ ] Rich commit message following PR template
- [ ] Tests still pass
- [ ] No merge conflicts
- [ ] History is clean and tells a story

You have complete autonomy to execute this rebase. Focus on creating a beautiful, clean commit that tells the story of this agent persona and worker isolation work.