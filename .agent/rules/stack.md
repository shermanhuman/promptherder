# Technology Stack

## Version Matrix

| Technology | Version | Status                 |
| ---------- | ------- | ---------------------- |
| Go         | 1.25.x  | Current (go.mod: 1.25) |
| doublestar | 4.6.x   | Glob pattern matching  |

## Key Constraints

- **Go version**: Minimum 1.25 (uses generics, enhanced error handling)
- **Standard library preferred**: Minimize external dependencies
- **Embedded FS**: Use `embed` package for Compound V resources
- **Atomic writes**: All file writes must use `AtomicWriter` pattern

## Never Do

- Never use external logging libraries (use stdlib `log/slog`)
- Never use reflection for core operations (keep it simple and fast)
- Never write files directly without atomic operations
- Never introduce dependencies for functionality available in stdlib
- Never use `os.Create` directly (use `AtomicWriter` or helper functions)
