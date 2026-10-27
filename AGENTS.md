# Agent Instructions

This is the PrintMaster product website, published at printmaster.work. The application lives in **Printmaster-Org/printmaster** (`../printmaster`), and the canonical documentation lives in **Printmaster-Org/docs.printmaster.work** (`../docs.printmaster.work`).

## Documentation impact (required check)

The canonical public user/operator documentation lives in **Printmaster-Org/docs.printmaster.work**, normally at `../docs.printmaster.work`. Assess documentation impact for every change; not every website or code change needs a documentation edit.

- Update canonical pages when a product-website change alters a user-facing task, supported behavior, downloads, releases, setup/configuration, deployment, or other guidance users and operators rely on. Keep the update focused on the changed reader need.
- Internal refactors, test-only changes, and build/CI maintenance normally need no docs-site edit when public behavior and instructions are unchanged. Keep this repository's README and deployment notes accurate when their own developer/operator instructions change.
- Link to canonical guides instead of duplicating them. The public docs site is not a place for implementation diaries, commit-by-commit audit notes, internal backlogs, or engineering progress reports.
- Follow the docs repository's [README](../docs.printmaster.work/README.md) for editing and validation when a docs change is needed. In the completion summary, state pages updated or briefly explain why no public-doc change was needed. Link paired implementation/docs commits or PRs when both changed.
- If the docs checkout is unavailable and a reader-facing update is required, report the blocker and exact pages needing updates; do not silently leave public guidance stale.

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
