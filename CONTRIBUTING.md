# Contributing to SupportFlow

Thank you for helping build SupportFlow. The project keeps its contribution process lightweight while the MVP is taking shape.

## Before You Start

- Search existing issues before opening a new one.
- Open an issue before substantial feature, architecture, or behavior changes.
- Report security problems privately according to [SECURITY.md](SECURITY.md).
- Keep changes focused; unrelated cleanup belongs in a separate pull request.

## Development Flow

1. Fork the repository and create a branch from `main`.
2. Make one coherent change with appropriate documentation and tests.
3. Use Conventional Commits.
4. Open a pull request describing the problem, solution, validation, and compatibility impact.

Suggested branch names include `feat/agent-state-machine`, `fix/ticket-idempotency`, and `docs/database-design`.

## Commit Messages

Use the following format:

```text
<type>(<scope>): <description>
```

Common types are `feat`, `fix`, `docs`, `refactor`, `test`, and `chore`.

Examples:

```text
feat(agent): implement run state machine
fix(ticket): prevent duplicate creation
docs(api): define SSE event contract
test(tool): cover permission failures
```

## Pull Requests

A pull request should:

- Reference the related issue when one exists.
- Explain user-visible and architectural effects.
- Include tests or explain why tests are not applicable.
- Update relevant documentation and the changelog when needed.
- Avoid secrets, personal data, generated runtime data, and unrelated files.

Maintainers may request that a large pull request be split into independently reviewable changes.

## Current Stage

The repository is currently design-first. The accepted PRD and architecture documents are the source of scope. Database, API, task breakdown, implementation, testing, and deployment design follow in that order.

By participating, you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).
