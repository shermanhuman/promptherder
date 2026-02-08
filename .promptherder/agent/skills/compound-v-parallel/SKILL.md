---
name: compound-v-parallel
description: Analyzes task dependencies and groups independent steps into parallel batches. Use when executing multi-step plans or performing research across multiple sources.
---

# Parallel Execution Skill

## When to use this skill

- executing a multi-step plan with independent steps
- researching multiple topics or URLs simultaneously
- creating multiple independent files
- running multiple independent commands

## Dependency analysis

1. List all steps / tool calls needed.
2. For each pair, check: does step B depend on output of step A?
   - Same file? → sequential.
   - B reads A's output? → sequential.
   - No overlap? → **parallel**.
3. Group independent steps into **batches**.

## Examples of parallelizable work

- Multiple web searches for different topics
- Multiple file reads on different files
- Multiple file writes for unrelated files
- Multiple commands that don't depend on each other

## When NOT to parallelize

- Steps that modify the same file
- Steps where output of one feeds into another
- Sequential build/test chains (build → test → deploy)
