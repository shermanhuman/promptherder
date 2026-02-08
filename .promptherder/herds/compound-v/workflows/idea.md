---
description: Add an idea to the future tasks list. Lightweight, no ceremony. Use any time.
---

// turbo-all

# Idea

Append the user's idea to `.promptherder/future-tasks.md`.

## Steps

1. Read `.promptherder/future-tasks.md` if it exists.

2. If it doesn't exist, create it with a header:

```markdown
# Future Tasks

Ideas deferred for later. Added by `/idea` or during `/plan`.
```

3. Append the idea as a checklist item:

```markdown
- [ ] <idea> — <brief context if provided> _(added: <YYYY-MM-DD>)_
```

4. Confirm to the user: **"Added to future tasks."** with the line that was added.

That's it. No planning, no research, no approval. Just append and confirm.
