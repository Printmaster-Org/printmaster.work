# Agent Instructions

This is the PrintMaster product website, published at printmaster.work. The application lives in **Printmaster-Org/printmaster** (`../printmaster`), and the canonical documentation lives in **Printmaster-Org/docs.printmaster.work** (`../docs.printmaster.work`).

## Commit messages (required)

Use [Conventional Commits](https://www.conventionalcommits.org/). The `.githooks/commit-msg` hook rejects other subjects. Enable it once per clone with `git config core.hooksPath .githooks`, and never bypass it with `--no-verify`.

```
type(scope): imperative summary, lowercase, no trailing period

Why the change was needed and what it changes.

Co-authored-by: ...   (trailers last)
```

- Types: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `build`, `ci`, `chore`, `style`, `revert`, `ux`. Add `!` after the type or scope for breaking changes.
- Scope is optional, lowercase, and names the area touched, for example `downloads`, `releases`, `web`, `analytics`, `docker`, `ci`, or `deps`.
- Keep the subject at 100 characters or fewer, with a blank line before the body.
