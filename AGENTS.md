# Agent Instructions

This is the PrintMaster product website, published at printmaster.work. The application lives in **Printmaster-Org/printmaster** (`../printmaster`), and the canonical documentation lives in **Printmaster-Org/docs.printmaster.work** (`../docs.printmaster.work`).

## Keep the docs site in sync (required)

- Review documentation impact for every feature, fix, removal, and deprecation. Changes to website behavior, downloads, releases, deployment, configuration, or UI must update the relevant canonical pages under `../docs.printmaster.work/content/` in the same task.
- Keep source-local READMEs and examples accurate, but link to the canonical docs rather than duplicating full guides. Follow the docs repository's [README](../docs.printmaster.work/README.md) for editing and validation.
- Link implementation and documentation commits or PRs, and report the updated pages and validation. If no documentation change is needed, explicitly explain why.
- If the docs checkout is unavailable, report the blocker and exact pages needing updates; do not silently leave docs stale or report the task as complete.

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
