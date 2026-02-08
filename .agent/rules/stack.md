# Tech Stack

| Technology | Version | Notes         |
| ---------- | ------- | ------------- |
| Go         | 1.25.x  | Main language |
| doublestar | v4.6.x  | Glob matching |

## Constraints

- **Standard Library**: Prefer `stdlib` over external deps where possible.
- **FS Abstraction**: Use `io/fs.FS` for file system interactions (testability).
- **Testing**:
  - `t.Parallel()` in every test function.
  - `t.TempDir()` for file I/O tests.
  - No global state mutation (`os.Chdir`).
- **Error Handling**: Use `errors.Is` and `fmt.Errorf("%w")`.
- **Logging**: Use `log/slog`.
