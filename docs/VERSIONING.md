# Versioning

This project uses automated Semantic Versioning driven by commit messages. You don’t need to manually edit VERSION, run scripts, or create tags. After a successful push to main, CI determines the next version from commit messages and publishes it.

At a glance
- Source of truth: The VERSION file in the repo root (maintained by CI).
- Automation trigger: Pushes to main after CI jobs succeed.
- Tagging: CI commits the new VERSION and creates/pushes a tag vX.Y.Z.
- No bump: If there are no semantic changes, no new version is created.

How it works
1. CI runs lint, test, build, and integration-test on every push/PR.
2. On push to main, if all the above jobs succeed, the versioning step:
   - Looks at commit messages since the last tag (or falls back to the whole history if no tag exists).
   - Determines the bump according to Conventional Commits.
   - Updates VERSION, commits chore(release): vX.Y.Z [skip ci], creates tag vX.Y.Z, and pushes both to main.

What bumps the version
The CI detects the highest-impact change present since the last tag:

- Major (X.y.z): Any commit containing BREAKING CHANGE in the body or using the ! syntax, for example:
  - feat!: drop support for legacy mode
  - refactor!: migrate to new API
  - Body includes: BREAKING CHANGE: details here

- Minor (x.Y.z): Any commit starting with:
  - feat:

- Patch (x.y.Z): Any commit starting with:
  - fix:
  - perf:
  - refactor:

- No bump: Commits that don’t match the above (e.g., chore:, docs:, test:, ci:) do not bump the version.

Notes
- Multiple types across commits: The highest level wins (major > minor > patch).
- First run behavior:
  - If tags exist, the base is the latest tag.
  - Else, if a VERSION file exists, that is used as the base.
  - Else, base starts at 0.0.0.
- Go modules: If you publish a v2+ and this repo is used as a Go module, you might need to update the module path in go.mod (e.g., /v2). CI does not do this automatically.

Practical guidance

Use Conventional Commits on PR titles or the merge commit message:
- feat: add new subcommand → minor bump
- fix: correct panic on empty config → patch bump
- feat!: remove deprecated flags → major bump
- docs: update README → no bump

Recommended merge strategy
- Squash and merge: Set the squash commit title/message to a proper Conventional Commit so CI can infer the correct bump.

Do not
- Manually edit VERSION for releases (CI owns it).
- Manually create tags for releases (CI creates vX.Y.Z).
- Depend on previous manual scripts/version.sh or version-bump Makefile targets (removed/obsolete).

Viewing the current version
- VERSION file in the repository root.
- Binary output: ./forgor version (includes ldflags set during build).

Troubleshooting

No bump happened:
- Was it a push to main? Versioning only runs on pushes to main.
- Did all CI jobs (lint, test, build, integration-test) pass?
- Did the commits since the last tag include feat:, fix:, perf:, refactor:, or a breaking change marker (! or BREAKING CHANGE)?
- If you didn’t intend to release, that’s fine—non-semantic commits won’t bump.

The bump level was unexpected:
- Check the exact commit message that landed on main (especially the final squash/merge commit message).
- For major bumps, ensure you either use feat!: or include BREAKING CHANGE in the commit body.

Future enhancements (optional)
- Automated changelog generation and GitHub Releases on tag push.
- Commit linting to enforce Conventional Commits on PRs.

This system keeps versioning consistent, predictable, and hands-off—focus on writing clear commit messages and the CI will take care of the rest.