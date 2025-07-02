# Versioning and Release Management

This document explains the versioning and release management system for `forgor`.

## Overview

`forgor` uses [Semantic Versioning (SemVer)](https://semver.org/) for version management:

- **MAJOR.MINOR.PATCH** format (e.g., `1.2.3`)
- **Prerelease versions** with identifiers (e.g., `1.2.3-alpha.1`, `1.2.3-beta.2`)

## Version Enforcement

🚨 **Important**: Every push/PR to the `main` branch **must** include a version bump. The CI will fail if the version isn't increased.

## Quick Start

### 1. Bump Version Types

```bash
# Patch version (1.0.0 → 1.0.1) - for bug fixes
make version-bump-patch

# Minor version (1.0.1 → 1.1.0) - for new features
make version-bump-minor

# Major version (1.1.0 → 2.0.0) - for breaking changes
make version-bump-major

# Prerelease (1.0.0 → 1.0.1-alpha.1) - for testing
make version-bump-prerelease
```

### 2. Create Releases

For a complete release with changelog, commit, and tag:

```bash
# Create a patch release
make release-patch

# Create a minor release
make release-minor

# Create a major release
make release-major

# Create a prerelease
make release-prerelease
```

### 3. Push to GitHub

```bash
# Push changes and tags
git push origin main && git push origin --tags
```

## Versioning Script

The `scripts/version.sh` script provides comprehensive version management:

### Commands

```bash
# Show current version
./scripts/version.sh current

# Bump versions
./scripts/version.sh bump patch
./scripts/version.sh bump minor
./scripts/version.sh bump major
./scripts/version.sh bump prerelease [identifier]

# Set specific version
./scripts/version.sh set 2.1.0

# Create git tag for current version
./scripts/version.sh tag

# Full release workflow
./scripts/version.sh release patch
```

### Examples

```bash
# Basic version bumping
$ ./scripts/version.sh current
0.1.0

$ ./scripts/version.sh bump patch
ℹ️  Current version: 0.1.0
ℹ️  New version: 0.1.1
✅ Updated VERSION to 0.1.1

# Prerelease with custom identifier
$ ./scripts/version.sh bump prerelease beta
ℹ️  Current version: 0.1.1
ℹ️  New version: 0.1.2-beta.1
✅ Updated VERSION to 0.1.2-beta.1

# Full release workflow
$ ./scripts/version.sh release minor
ℹ️  Creating release 0.2.0
ℹ️  Current version: 0.1.2-beta.1
ℹ️  New version: 0.2.0
✅ Updated VERSION to 0.2.0
✅ Updated CHANGELOG.md
✅ Created tag v0.2.0
ℹ️  To push changes and tags, run:
ℹ️    git push origin main && git push origin v0.2.0
```

## GitHub Actions Workflows

### Version Check (`.github/workflows/version-check.yml`)

Runs on every push/PR to `main`:

- ✅ Validates `VERSION` file exists and has valid SemVer format
- ✅ Ensures version is bumped for PRs to main
- ✅ Compares versions to ensure proper incrementing
- ❌ **Fails the build** if version isn't bumped

### Automated Releases (`.github/workflows/release.yml`)

Triggers when version tags are pushed (e.g., `v1.2.3`):

- 🔨 Builds binaries for multiple platforms (Linux, macOS, Windows)
- 📦 Creates GitHub release with changelog
- 🔗 Attaches binaries and checksums
- 🏷️ Marks prereleases appropriately

## Makefile Integration

The `Makefile` includes version information in builds:

```bash
# Show version info
make version

# Validate VERSION file
make version-check

# Build with version info embedded
make build

# Build for all platforms
make build-all
```

## File Structure

```
├── VERSION                           # Current version (e.g., "0.1.0")
├── CHANGELOG.md                      # Generated/maintained changelog
├── scripts/
│   └── version.sh                    # Versioning script
├── .github/workflows/
│   ├── version-check.yml             # Enforces version bumps
│   └── release.yml                   # Automated releases
└── docs/
    └── VERSIONING.md                 # This file
```

## Workflow Examples

### Bug Fix Release

```bash
# Fix bug in code
git add .
git commit -m "fix: resolve memory leak in command processing"

# Bump patch version
make version-bump-patch

# Create release
make release-patch

# Push everything
git push origin main && git push origin --tags
```

### Feature Release

```bash
# Add new feature
git add .
git commit -m "feat: add support for custom completion scripts"

# Bump minor version
make version-bump-minor

# Create release
make release-minor

# Push everything
git push origin main && git push origin --tags
```

### Prerelease Testing

```bash
# Add experimental feature
git add .
git commit -m "feat: experimental AI command suggestions"

# Create prerelease
make version-bump-prerelease

# Test and iterate
make version-bump-prerelease  # Creates alpha.2, alpha.3, etc.

# When ready, create full release
make release-minor
```

## Best Practices

### 1. **Always Bump Before Merging**

Never merge to `main` without bumping the version. The CI will catch this and fail.

### 2. **Use Conventional Commits**

- `fix:` → patch version
- `feat:` → minor version
- `feat!:` or `BREAKING CHANGE:` → major version

### 3. **Update Changelog**

The release commands automatically update `CHANGELOG.md`, but you can edit it for more detailed notes.

### 4. **Test Prereleases**

Use prereleases for testing major changes before full release.

### 5. **Semantic Versioning**

- **Patch** (0.0.X): Bug fixes, no API changes
- **Minor** (0.X.0): New features, backward compatible
- **Major** (X.0.0): Breaking changes

## Troubleshooting

### CI Fails: "Version must be bumped"

```bash
# Check current version
./scripts/version.sh current

# Bump appropriate version
make version-bump-patch  # or minor/major

# Commit and push
git add VERSION && git commit -m "chore: bump version" && git push
```

### Invalid Version Format

```bash
# Check version format
make version-check

# If invalid, set correct version
./scripts/version.sh set 1.0.0
```

### Missing VERSION File

```bash
# Create VERSION file
echo "0.1.0" > VERSION
git add VERSION && git commit -m "chore: add VERSION file"
```

## Advanced Usage

### Custom Prerelease Identifiers

```bash
./scripts/version.sh bump prerelease rc     # 1.0.0 → 1.0.1-rc.1
./scripts/version.sh bump prerelease beta   # 1.0.0 → 1.0.1-beta.1
./scripts/version.sh bump prerelease nightly # 1.0.0 → 1.0.1-nightly.1
```

### Manual Version Setting

```bash
# Set specific version (must be greater than current)
./scripts/version.sh set 2.0.0-beta.5
```

### Build Information

The version command shows detailed build information:

```bash
$ ./forgor version

═══════════════ FORGOR VERSION INFO ═══════════════
┌──────────────┬─────────────────────────────────────┐
│ Info         │ Value                               │
├──────────────┼─────────────────────────────────────┤
│ Version      │ 0.1.0                              │
│ Git Commit   │ abc123def456                        │
│ Build Date   │ 2024-01-15 14:30:25 UTC           │
│ Go Version   │ go1.21.0                           │
│ Platform     │ darwin/arm64                        │
└──────────────┴─────────────────────────────────────┘
Executable: /usr/local/bin/forgor

💡 Check for updates at: https://github.com/siutan/forgor/releases
```

This system ensures consistent, automated, and enforced version management across the entire project lifecycle.
