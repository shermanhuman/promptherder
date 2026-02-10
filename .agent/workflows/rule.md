---
description: Add a hard rule to the project's always-on prompt. Lightweight, no ceremony. Use any time.
---

// turbo-all

# Rule

Append the user's rule to `.promptherder/hard-rules.md`.

**What this does:** Adds a rule that the AI must always follow in this project. Hard rules are injected into every session automatically. Use this for project-specific standards, constraints, or guardrails (e.g., "never use eval", "all API routes require auth", "prefer composition over inheritance").

## Steps

1. If the user didn't provide a rule (e.g. just typed `/rule` with nothing after), prompt them:

> **What rule should always be enforced in this project?** Describe it as a clear, actionable statement.

2. Read `.promptherder/hard-rules.md` if it exists.

3. If it doesn't exist, create it with a header:

```markdown
---
trigger: always_on
---

# Hard Rules

Project-level rules that are always active. Added by `/rule` or manually.
```

4. Append the rule as a bullet point:

```markdown
- <rule>
```

5. Confirm to the user: **"Added hard rule."** with the line that was added.

That's it. No planning, no research, no approval. Just append and confirm.
