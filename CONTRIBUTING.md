# Contributing to godl

## Development Guidelines

### Code Style

1. **Comments must be in English only**
   - All code comments, documentation, and commit messages should be in English
   - Use clear and concise language

2. **Follow Go conventions**
   - Use `gofmt` for formatting
   - Follow [Effective Go](https://golang.org/doc/effective_go.html)
   - Use meaningful variable and function names

3. **Error handling**
   - Always handle errors explicitly
   - Wrap errors with context using `fmt.Errorf`
   - Create custom error types when appropriate

4. **Testing**
   - Write tests for all new features
   - Maintain test coverage above 80%
   - Use table-driven tests where appropriate

### For AI Assistants (Claude, GitHub Copilot, etc.)

When generating code for this project:

1. **All comments and documentation must be in English**
2. **Follow Go idioms and best practices**
3. **Include comprehensive error handling**
4. **Add unit tests for any new functions**
5. **Use descriptive variable names, not single letters**
6. **Keep functions small and focused (< 50 lines)**
7. **Add godoc comments for all exported functions**

### Commit Messages

Follow the conventional commits format:
- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation changes
- `test:` for test additions/changes
- `refactor:` for code refactoring
- `chore:` for maintenance tasks

Example:
```
feat: add concurrent download support

- Implement chunk-based downloading
- Add progress tracking for each chunk
- Support up to 10 concurrent connections
```
