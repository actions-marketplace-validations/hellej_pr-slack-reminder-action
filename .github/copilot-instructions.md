# Copilot Instructions for PR Slack Reminder Action

## Code Style

- Help the author to learn writing idiomatic Go code (the author is new to Go programming language)
- Encourage the use of Go modules and proper package structure
- Promote the use of clear and descriptive naming conventions for variables and functions
- Advocate for the use of idiomatic error handling patterns (e.g., returning errors instead of panicking)
- Encourage writing unit/integration tests for new features and maintaining high test coverage

## AI Code Generation

- Ask for more instructions and context if the prompt is unclear or lacks detail
- Do not add comments explaining code, unless it is complex or non-obvious
- Aim for simplicity and clarity in generated code

### Testing Strategy & Practices

- **Test-Driven Development**: Use TDD approach when adding new features - write tests first, then implement the functionality whenever possible
- **Pragmatic Testing**: Balance thoroughness with practicality - focus on critical paths and user-facing behavior
- **Test Public Interfaces**: Focus tests on client/user-facing features and functionality rather than implementation details to avoid brittle tests that break during refactoring
- **Readable Tests**: Write tests that are easy to understand with minimal comments - the test structure and assertions should be self-explanatory
- **Reduce Boilerplate**: Create test helpers for setup and assertions when duplication emerges - check for existing helpers in `testhelpers/` package
- **Test Coverage**: Maintain high test coverage while avoiding testing internal implementation details that may change during refactoring
- **Table-Driven Tests**: Use table-driven tests for functions with multiple input scenarios to reduce duplication and improve readability
- **Main Tests**: `cmd/pr-slack-reminder/main_test.go` contains integration tests using full pipeline with mocks
- **Test Helpers**: `testhelpers/confighelpers.go` provides `TestConfig` struct and `SetTestEnvironment()` for consistent test setup
- **Mock Clients**: `testhelpers/mock*client/` provide injectable dependencies for testing

### Development Commands

See the Makefile for all available commands.

```bash
# Run with env vars (see Makefile `run` target for pattern)
make run

# Test with coverage
make test-with-coverage

# Build a specific target
make build-darwin-amd64

# Validate inputs consistency
go run .github/scripts/check_inputs.go
```

## Architecture Overview

This is a GitHub Action written in Go that fetches PRs from GitHub repositories and sends a Slack reminder listing them. The architecture follows a clear data pipeline:

1. **Config** (`internal/config/`) - Parses GitHub Action inputs using environment variables with `INPUT_` prefix pattern
2. **GitHub Client** (`internal/apiclients/githubclient/`) - Fetches PR data and reviews, applies filtering
3. **PR Parser** (`internal/prparser/`) - Enriches PR data with Slack user mappings and metadata
4. **Message Content** (`internal/messagecontent/`) - Structures data for messaging
5. **Message Builder** (`internal/messagebuilder/`) - Constructs Slack Block Kit messages
6. **Slack Client** (`internal/apiclients/slackclient/`) - Sends messages

## Key Patterns

### Input Configuration

- All GitHub Action inputs are accessed via `utilities.GetInput()` which converts `input-name` to `INPUT_INPUT_NAME` env vars
- Repository-specific mappings use semicolon/newline-separated format: `"repo1: value1; repo2: value2"`
- JSON inputs (filters, mappings) are parsed with `DisallowUnknownFields()` for strict validation

### Repository Processing

- Multiple repositories supported via `config.Repositories` slice of `Repository` structs (`config.InputGithubRepositories`)
- If `config.InputGithubRepositories` is set, `config.EnvGithubRepository` is ignored
- Repository filters and prefixes are mapped by repository name (not full path)
- Each PR maintains its `Repository` field for context throughout the pipeline

### Error Handling

- Config validation uses `selectNonNilError()` to return first encountered error
- Filters validate mutual exclusivity (e.g., can't use both `authors` and `authors-ignore`)
- Missing required inputs fail fast with descriptive error messages

### Slack Message Construction

- Uses Slack Block Kit with `RichTextBlock` and `RichTextSection` elements
- PR titles are clickable links, prefixes are separate text elements before links
- Age indicators use emoji and bold styling for old PRs (`IsOldPR` field)

## File Relationships

- `action.yml` inputs must match constants in `internal/config/config.go`
- Test environment setup in `testhelpers/confighelpers.go` mirrors real config parsing
- The validation script `.github/scripts/check_inputs.go` ensures `action.yml` and config constants stay in sync

## Adding New Features

(remember to follow TDD if possible)

1. Add input to `action.yml`
2. Add constant to `internal/config/config.go`
3. Update `Config` struct and `GetConfig()` function
4. Update `testhelpers/confighelpers.go` for test support
5. Implement feature logic in appropriate pipeline stage
