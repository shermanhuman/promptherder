---
name: compound-v-tdd
description: Applies tests-first discipline (red/green/refactor) and adds regression tests for bugs. Use when implementing features, fixing bugs, or refactoring.
---

# TDD Skill

**Announce at start:** "Using TDD: writing test first for [behavior]."

## When to use this skill

- new features that can be unit tested
- bug fixes (always add a regression test if practical)
- refactors (protect behavior with tests first)

## Research

Before writing tests, `search_web` for:

- The project's testing framework best practices and latest patterns (scope to `stack.md` versions).
- Assertion styles and testing utilities available in the framework.

## Rules

- Prefer **red -> green -> refactor**.
- If tests are hard, still add **verification**: minimal repro script, integration test, or clear manual steps.
- Keep tests focused: one behavior per test where possible.
- Name tests by behavior, not implementation details.

## Process

1. Define the behavior change (what should be true after).
2. Write/adjust a test to capture it (make it fail first if possible).
3. Implement the minimal change to pass.
4. Refactor if needed (keep passing).
5. Run the relevant test suite + any linters **in parallel** where they are independent.

## Output requirements

When you change code, include:

- what tests you added/changed
- how to run them
- what they prove

## Bite-sized granularity

Each step should be one action (2-5 minutes):

1. Write the failing test — step
2. Run it to confirm it fails — step
3. Implement the minimal code to make it pass — step
4. Run tests to confirm they pass — step
5. Commit — step

Don't combine these. Each step is independently verifiable.
