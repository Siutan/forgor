# Development Guide

This guide covers the development workflow, tools, and best practices for contributing to the `forgor` CLI project.

## 📋 Table of Contents

- [Quick Start](#quick-start)
- [Development Workflow](#development-workflow)
- [Creating Pull Requests](#creating-pull-requests)
- [Code Quality](#code-quality)
- [Testing](#testing)
- [Version Management](#version-management)
- [Troubleshooting](#troubleshooting)

## 🚀 Quick Start

### Prerequisites

- **Go 1.20+**: [Download](https://golang.org/dl/)
- **Git**: [Download](https://git-scm.com/downloads)
- **GitHub CLI**: [Download](https://cli.github.com/) (for PR creation)
- **Make**: Usually pre-installed on Unix systems

### Setup

```bash
# Clone the repository
git clone https://github.com/YOURUSERNAME/forgor.git
cd forgor

# Install dependencies
make deps

# Build the project
make build

# Run tests
make test

# Check version info
make version-info
```

## 💻 Development Workflow

### 1. Create Feature Branch

```bash
# Always start from main
git checkout main
git pull origin main

# Create feature branch
git checkout -b feat/your-feature-name
# or
git checkout -b fix/issue-number
# or
git checkout -b chore/task-description
```

### 2. Make Changes

```bash
# Make your changes...

# Run quality checks frequently
make check-quality

# Test your changes
make test
./forgor --help  # Test CLI locally
```

### 3. Commit Changes

```bash
# Format code and run checks
make pre-commit

# Add and commit
git add .
git commit -m "feat: add awesome new feature"
```

### 4. Create Pull Request

**Option 1: Automated (Recommended)**

```bash
# Use the automated PR script
make create-pr -b "feat/your-feature" -t "Add awesome new feature"
```

**Option 2: Manual**

```bash
# Push branch
git push origin feat/your-feature

# Create PR manually
gh pr create --title "Add awesome new feature" --body "Description here"
```

## 🔄 Creating Pull Requests

### Automated PR Creation

The project includes a comprehensive PR creation script that handles all quality checks:

```bash
# Basic usage
scripts/create-pr.sh -b "feat/new-feature" -t "Add new feature"

# With description
scripts/create-pr.sh \
  -b "fix/bug-123" \
  -t "Fix critical bug" \
  -d "This fixes the issue where the app crashes on startup"

# Create draft PR
scripts/create-pr.sh \
  -b "wip/experimental" \
  -t "WIP: Experimental feature" \
  --draft

# Skip tests (not recommended)
scripts/create-pr.sh \
  -b "docs/update" \
  -t "Update documentation" \
  --skip-tests
```

### What the PR Script Does

✅ **Pre-flight Checks**

- Validates git repository state
- Checks GitHub CLI authentication
- Ensures you're not on main branch

✅ **Code Quality**

- **Auto-formats code** with `gofmt`
- **Runs linting** with `go vet`
- **Executes tests** with race detection
- **Builds project** to ensure compilation

✅ **Version Management**

- **Checks version bump** compared to main
- **Prompts for version bump** if needed
- **Supports all bump types** (patch/minor/major)

✅ **PR Creation**

- **Pushes branch** to remote
- **Creates GitHub PR** with proper template
- **Opens in browser** (optional)

### PR Script Options

| Option              | Description               | Example                     |
| ------------------- | ------------------------- | --------------------------- |
| `-b, --branch`      | Branch name (required)    | `-b "feat/new-feature"`     |
| `-t, --title`       | PR title (required)       | `-t "Add new feature"`      |
| `-d, --description` | PR description            | `-d "Detailed description"` |
| `--draft`           | Create as draft PR        | `--draft`                   |
| `--no-auto-fix`     | Don't auto-fix formatting | `--no-auto-fix`             |
| `--skip-tests`      | Skip running tests        | `--skip-tests`              |
| `--force`           | Skip safety checks        | `--force`                   |

## 🔍 Code Quality

### Automated Quality Checks

```bash
# Run all quality checks
make check-quality

# Individual checks
make fmt          # Format code
make vet          # Static analysis
make lint         # Advanced linting
make test         # Run tests
```

### Code Style Guidelines

**Go Code:**

- Use `gofmt` for formatting (automated)
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Write clear, self-documenting code
- Add comments for exported functions

**Commit Messages:**

```
type(scope): description

# Types: feat, fix, docs, style, refactor, test, chore
# Examples:
feat(cli): add new command for user management
fix(config): resolve parsing issue with YAML files
docs(readme): update installation instructions
```

### Pre-commit Hooks (Manual)

```bash
# Run before each commit
make pre-commit
```

This runs:

- Code formatting
- Linting
- Tests
- Version validation

## 🧪 Testing

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage
# Opens coverage.html in browser

# Run specific test
go test ./internal/utils -v

# Run with race detection
go test -race ./...
```

### Writing Tests

**Test Structure:**

```go
func TestFunctionName(t *testing.T) {
    // Setup
    input := "test input"
    expected := "expected output"

    // Execute
    result := FunctionName(input)

    // Assert
    if result != expected {
        t.Errorf("FunctionName(%s) = %s; want %s", input, result, expected)
    }
}
```

**Test Coverage Requirements:**

- Minimum **50%** coverage (enforced by CI)
- All new features must include tests
- Bug fixes should include regression tests

## 📈 Version Management

### Version Bumping

The project uses [Semantic Versioning](https://semver.org/):

```bash
# Patch (bug fixes): 1.0.0 → 1.0.1
make version-bump-patch

# Minor (new features): 1.0.0 → 1.1.0
make version-bump-minor

# Major (breaking changes): 1.0.0 → 2.0.0
make version-bump-major

# Prerelease: 1.0.0 → 1.0.1-alpha.1
make version-bump-prerelease
```

### When to Bump Version

**Required for main branch PRs:**

- Every PR to main must bump the version
- Version bump enforced by CI
- Choose appropriate level based on changes

**Version Types:**

- **Patch**: Bug fixes, documentation updates
- **Minor**: New features, enhancements
- **Major**: Breaking changes, API changes
- **Prerelease**: Alpha/beta releases

### Release Process

```bash
# Create release (bumps version and creates tag)
make release-patch   # or release-minor, release-major

# Manual release
scripts/version.sh bump patch
scripts/version.sh release
```

## 🛠️ Troubleshooting

### Common Issues

**❌ "Code is not properly formatted"**

```bash
# Fix automatically
make fmt
```

**❌ "Tests failed"**

```bash
# Run tests to see details
make test

# Fix failing tests
# Run specific test for debugging
go test ./path/to/package -v -run TestFunctionName
```

**❌ "Version hasn't been bumped"**

```bash
# For PRs to main, bump version
make version-bump-patch  # or minor/major
```

**❌ "GitHub CLI not authenticated"**

```bash
# Authenticate with GitHub
gh auth login
```

**❌ "golangci-lint not found"**

```bash
# Install golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

### Debug Commands

```bash
# Check git status
git status

# Check current version
make version-info

# Validate VERSION file
make version-check

# Check dependencies
make deps

# Clean and rebuild
make clean && make build
```

### CI/CD Issues

**Check CI Logs:**

1. Go to GitHub Actions tab
2. Click on failing job
3. Expand failing step
4. Review error messages

**Common CI Fixes:**

```bash
# Format code
make fmt
git add . && git commit -m "style: fix formatting"

# Fix tests
make test  # Run locally first
# Fix issues, then commit

# Bump version
make version-bump-patch
git add VERSION && git commit -m "chore: bump version"
```

## 📚 Useful Resources

### Makefile Targets

```bash
make help  # Show all available commands
```

### Key Commands

| Command              | Purpose                |
| -------------------- | ---------------------- |
| `make build`         | Build the binary       |
| `make test`          | Run tests              |
| `make check-quality` | Run all quality checks |
| `make pre-commit`    | Pre-commit validation  |
| `make create-pr`     | Create PR with checks  |
| `make version-info`  | Show version details   |

### Documentation

- [CI/CD Documentation](CI.md) - Complete CI pipeline guide
- [Versioning Documentation](VERSIONING.md) - Version management details
- [README](../readme.md) - Project overview and usage

### External Resources

- [Go Documentation](https://golang.org/doc/)
- [GitHub CLI Documentation](https://cli.github.com/manual/)
- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)

## 🎯 Best Practices

### Development Flow

1. **Always start from main** - `git checkout main && git pull`
2. **Create feature branch** - `git checkout -b feat/feature-name`
3. **Make small commits** - Frequent, focused commits
4. **Run tests often** - `make test` during development
5. **Use quality checks** - `make check-quality` before commits
6. **Automated PR creation** - Use `scripts/create-pr.sh`

### Code Quality

1. **Write tests first** - TDD approach when possible
2. **Keep functions small** - Single responsibility principle
3. **Document public APIs** - Clear comments for exported functions
4. **Handle errors properly** - Don't ignore error returns
5. **Use meaningful names** - Clear variable and function names

### Performance

1. **Profile when needed** - Use `go test -bench=.`
2. **Avoid premature optimization** - Profile first, optimize second
3. **Memory awareness** - Consider memory usage patterns
4. **Concurrent safety** - Use proper synchronization

This development guide ensures high-quality contributions and smooth collaboration. Happy coding! 🚀
