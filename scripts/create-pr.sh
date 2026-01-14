#!/bin/bash

# create-pr.sh - Automated PR creation script with quality checks
# This script ensures all code quality requirements are met before creating a PR

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Icons
CHECK="✅"
CROSS="❌"
WARNING="⚠️"
INFO="ℹ️"
ROCKET="🚀"
GEAR="⚙️"
TEST="🧪"
FORMAT="📝"
VERSION="📈"

# Default values
BRANCH=""
BASE_BRANCH="main"
TITLE=""
DESCRIPTION=""
DRAFT=false
AUTO_FIX=true
SKIP_TESTS=false
FORCE=false
OPEN_BROWSER=false
CLEAN_WORKTREE=true
FORMAT_CHANGED=false

# Helper functions
print_header() {
    echo -e "\n${BLUE}═══════════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════════${NC}\n"
}

print_step() {
    echo -e "${CYAN}${GEAR} $1${NC}"
}

print_success() {
    echo -e "${GREEN}${CHECK} $1${NC}"
}

print_error() {
    echo -e "${RED}${CROSS} $1${NC}" >&2
}

print_warning() {
    echo -e "${YELLOW}${WARNING} $1${NC}"
}

print_info() {
    echo -e "${BLUE}${INFO} $1${NC}"
}

# Help function
show_help() {
    cat << EOF
${CYAN}create-pr.sh${NC} - Automated PR creation with quality checks

${YELLOW}USAGE:${NC}
    ./scripts/create-pr.sh [OPTIONS]

${YELLOW}OPTIONS:${NC}
    -b, --branch BRANCH        Branch name (defaults to current)
    --base BRANCH              Base branch (default: main)
    -t, --title TITLE          PR title (defaults to last commit subject)
    -d, --description DESC     PR description
    --draft                    Create as draft PR
    --open                     Open PR in browser after creation
    --no-auto-fix              Don't automatically fix formatting issues
    --skip-tests               Skip running tests (not recommended)
    --force                    Skip some safety checks
    -h, --help                 Show this help

${YELLOW}EXAMPLES:${NC}
    # Basic PR creation
    ./scripts/create-pr.sh -t "Add awesome new feature"
    
    # PR with description
    ./scripts/create-pr.sh -t "Fix critical bug" -d "Fixes issue #123"
    
    # Draft PR
    ./scripts/create-pr.sh -t "WIP: Experimental feature" --draft

${YELLOW}QUALITY CHECKS:${NC}
    ${CHECK} Code formatting (gofmt -s)
    ${CHECK} Linting (go vet + golangci-lint if available)
    ${CHECK} Tests pass
    ${CHECK} Git status clean
    ${CHECK} Branch is up to date

${YELLOW}COMMIT MESSAGE GUIDELINES:${NC}
    Versioning is automatic and inferred from commit messages on main.
    Use Conventional Commits to control the next version:
    
    feat: add feature             # minor
    fix: correct issue            # patch
    feat!: breaking change        # major
    (or include "BREAKING CHANGE:" in the body)

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -b|--branch)
                BRANCH="$2"
                shift 2
                ;;
            --base)
                BASE_BRANCH="$2"
                shift 2
                ;;
            -t|--title)
                TITLE="$2"
                shift 2
                ;;
            -d|--description)
                DESCRIPTION="$2"
                shift 2
                ;;
            --draft)
                DRAFT=true
                shift
                ;;
            --open)
                OPEN_BROWSER=true
                shift
                ;;
            --no-auto-fix)
                AUTO_FIX=false
                shift
                ;;
            --skip-tests)
                SKIP_TESTS=true
                shift
                ;;
            --force)
                FORCE=true
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done
}

# Validate required arguments
validate_args() {
    if [[ -z "$TITLE" ]]; then
        TITLE=$(git log -1 --pretty=%s 2>/dev/null || true)
        if [[ -z "$TITLE" ]]; then
            print_error "PR title is required. Use -t or --title"
            exit 1
        fi
        print_info "Using latest commit subject as PR title: $TITLE"
    fi

    if [[ -z "$BRANCH" ]]; then
        BRANCH="$CURRENT_BRANCH"
    fi

    if [[ "$BRANCH" != "$CURRENT_BRANCH" ]]; then
        print_error "Current branch is '$CURRENT_BRANCH' but '--branch' is '$BRANCH'"
        print_info "Switch branches or omit --branch to use the current branch"
        exit 1
    fi
}

# Check if we're in a git repository
check_git_repo() {
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        print_error "Not in a git repository"
        exit 1
    fi
}

# Check if GitHub CLI is installed
check_gh_cli() {
    if ! command -v gh &> /dev/null; then
        print_error "GitHub CLI (gh) is not installed"
        print_info "Install it from: https://cli.github.com/"
        exit 1
    fi
}

# Check if we're authenticated with GitHub
check_gh_auth() {
    if ! gh auth status &> /dev/null; then
        print_error "Not authenticated with GitHub CLI"
        print_info "Run: gh auth login"
        exit 1
    fi
}

# Check git status
check_git_status() {
    print_step "Checking git status..."
    
    # Check if we have uncommitted changes
    if [[ -n $(git status --porcelain) ]]; then
        print_warning "You have uncommitted changes"
        CLEAN_WORKTREE=false
        if [[ "$FORCE" == "false" ]]; then
            print_info "Commit your changes first, or use --force to continue"
            exit 1
        fi
        print_warning "Proceeding with dirty working tree (changes won't be pushed)"
    fi
    
    # Check current branch
    CURRENT_BRANCH=$(git branch --show-current)
    if [[ -z "$CURRENT_BRANCH" ]]; then
        print_error "Detached HEAD state. Check out a branch first"
        exit 1
    fi
    if [[ "$CURRENT_BRANCH" == "main" ]]; then
        print_error "Cannot create PR from main branch"
        print_info "Create a feature branch first: git checkout -b your-feature-branch"
        exit 1
    fi
    
    print_success "Git status OK (current branch: $CURRENT_BRANCH)"
}

# Check if branch exists and push if needed
check_and_push_branch() {
    print_step "Checking branch status..."
    
    # Check if branch exists on remote
    if ! git ls-remote --heads origin "$BRANCH" | grep -q "$BRANCH"; then
        print_info "Branch doesn't exist on remote, will push after checks"
        NEED_PUSH=true
    else
        print_info "Branch exists on remote"
        git fetch --quiet origin "$BRANCH"

        COUNTS=$(git rev-list --left-right --count "origin/$BRANCH...$BRANCH")
        BEHIND=${COUNTS%% *}
        AHEAD=${COUNTS##* }

        if [[ "$BEHIND" -gt 0 && "$AHEAD" -gt 0 ]]; then
            print_error "Your branch has diverged from origin/$BRANCH"
            print_info "Please rebase or merge before creating a PR"
            exit 1
        fi

        if [[ "$BEHIND" -gt 0 ]]; then
            print_error "Your branch is behind origin/$BRANCH"
            print_info "Please pull or rebase before creating a PR"
            exit 1
        fi

        if [[ "$AHEAD" -gt 0 ]]; then
            print_info "Local branch is ahead of remote, will push updates"
            NEED_PUSH=true
        else
            NEED_PUSH=false
        fi
    fi
}

# Format code
format_code() {
    print_step "Checking code formatting..."
    
    # Check if code needs formatting
    UNFORMATTED=$(gofmt -l -s . 2>/dev/null || true)
    
    if [[ -n "$UNFORMATTED" ]]; then
        print_warning "Code needs formatting:"
        echo "$UNFORMATTED"
        
        if [[ "$AUTO_FIX" == "true" ]]; then
            print_step "Auto-fixing formatting..."
            gofmt -w -s .
            print_success "Code formatted successfully"
            FORMAT_CHANGED=true
        else
            print_error "Please format your code with: make fmt"
            exit 1
        fi
    else
        print_success "Code is properly formatted"
    fi
}

# Commit gofmt changes if we started clean
commit_formatting_changes() {
    if [[ "$FORMAT_CHANGED" == "false" ]]; then
        return
    fi

    if [[ "$CLEAN_WORKTREE" == "false" ]]; then
        print_error "Formatting changes applied on top of uncommitted work"
        print_info "Please review and commit formatting changes, then rerun"
        exit 1
    fi

    git add -u
    git commit -m "style: gofmt"
    print_success "Committed formatting changes"
}

# Run linting
run_linting() {
    print_step "Running linting checks..."
    
    if ! make lint; then
        print_error "Linting failed"
        exit 1
    fi
    
    print_success "Linting checks passed"
}

# Run tests
run_tests() {
    if [[ "$SKIP_TESTS" == "true" ]]; then
        print_warning "Skipping tests (not recommended for production)"
        return
    fi
    
    print_step "Running tests..."
    
    if ! make test; then
        print_error "Tests failed"
        print_info "Fix the failing tests before creating PR"
        exit 1
    fi
    
    print_success "All tests passed"
}

# Check version bump
# Version bump checks removed; CI infers version from Conventional Commits.

# Build project to ensure it compiles
build_project() {
    print_step "Building project..."
    
    if ! make build; then
        print_error "Project build failed"
        exit 1
    fi
    
    print_success "Project builds successfully"
}

# Push branch if needed
push_branch() {
    if [[ "$NEED_PUSH" == "true" ]]; then
        print_step "Pushing branch to remote..."
        
        git push origin "$CURRENT_BRANCH"
        print_success "Branch pushed to origin/$CURRENT_BRANCH"
    fi
}

# Create PR
create_pr() {
    print_step "Creating pull request..."
    
    # Prepare PR body
    PR_BODY="$DESCRIPTION"
    
    if [[ -z "$PR_BODY" ]]; then
        PR_BODY="## Summary

Changes in this PR:
- 

## Testing
- [ ] Tests pass locally
- [ ] Manual testing completed

## Checklist
- [x] Code is formatted
- [x] Tests pass
- [x] Commit message follows Conventional Commits (feat/fix/feat! or BREAKING CHANGE)
- [x] Documentation updated (if needed)"
    fi
    
    # Create PR command
    PR_CMD=(gh pr create --title "$TITLE" --body "$PR_BODY" --base "$BASE_BRANCH" --head "$BRANCH")
    if [[ "$DRAFT" == "true" ]]; then
        PR_CMD+=(--draft)
    fi

    # Execute PR creation
    if "${PR_CMD[@]}"; then
        print_success "Pull request created successfully!"
        
        # Get PR URL
        PR_URL=$(gh pr view --json url --jq .url)
        print_info "PR URL: $PR_URL"
        
        if [[ "$OPEN_BROWSER" == "true" ]]; then
            gh pr view --web
        fi
    else
        print_error "Failed to create pull request"
        exit 1
    fi
}

# Show summary
show_summary() {
    print_header "PR CREATION SUMMARY"
    
    echo -e "${GREEN}${ROCKET} Successfully created PR:${NC}"
    echo -e "  Title: $TITLE"
    echo -e "  Branch: $BRANCH → $BASE_BRANCH"
    echo -e "  Draft: $DRAFT"
    echo ""
    
    echo -e "${BLUE}${INFO} Next steps:${NC}"
    echo -e "  1. Wait for CI checks to complete"
    echo -e "  2. Request reviews from team members"
    echo -e "  3. Address any feedback"
    echo -e "  4. Merge when approved and CI passes"
    echo ""
    
    print_info "Monitor your PR: gh pr view"
}

# Main execution
main() {
    print_header "FORGOR PR CREATION ASSISTANT"
    
    parse_args "$@"
    
    # Pre-flight checks
    check_git_repo
    check_gh_cli
    check_gh_auth
    check_git_status
    validate_args
    
    # Code quality checks
    format_code
    commit_formatting_changes
    run_linting
    run_tests
    # Version bump check removed; CI handles versioning based on Conventional Commits
    build_project

    check_and_push_branch
    
    # Push and create PR
    push_branch
    create_pr
    
    # Summary
    show_summary
    
    print_success "All done! Your PR is ready for review."
}

# Run main function with all arguments
main "$@" 
