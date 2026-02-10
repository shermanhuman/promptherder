---
description: Add an idea to the future tasks list. Lightweight, no ceremony. Use any time.
---

// turbo-all

# Idea

Append the user's idea to `.promptherder/future-tasks.md`.

**What this does:** Saves an idea for later so you don't lose it. Ideas are stored as a checklist in `.promptherder/future-tasks.md` and can be pulled into a `/plan` when the time is right.

## Steps

1. If the user didn't provide an idea (e.g. just typed `/idea` with nothing after), prompt them:

> **What idea would you like to save for later?** Just describe it in a sentence or two.

2. Read `.promptherder/future-tasks.md` if it exists.

3. If it doesn't exist, create it with a header:

```markdown
# Future Tasks

Ideas deferred for later. Added by `/idea` or during `/plan`.
```

4. Append the idea as a checklist item:

```markdown
- [ ] <idea> — <brief context if provided> _(added: <YYYY-MM-DD>)_
```

5. Confirm to the user: **"Added to future tasks."** with the line that was added.

That's it. No planning, no research, no approval. Just append and confirm.
