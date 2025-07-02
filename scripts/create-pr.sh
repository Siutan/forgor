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
TITLE=""
DESCRIPTION=""
DRAFT=false
AUTO_FIX=true
SKIP_TESTS=false
FORCE=false

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
    -b, --branch BRANCH        Target branch name (required)
    -t, --title TITLE          PR title (required)
    -d, --description DESC     PR description
    --draft                    Create as draft PR
    --no-auto-fix             Don't automatically fix formatting issues
    --skip-tests              Skip running tests (not recommended)
    --force                   Skip some safety checks
    -h, --help                Show this help

${YELLOW}EXAMPLES:${NC}
    # Basic PR creation
    ./scripts/create-pr.sh -b "feat/new-feature" -t "Add awesome new feature"
    
    # PR with description
    ./scripts/create-pr.sh -b "fix/bug-123" -t "Fix critical bug" -d "Fixes issue #123"
    
    # Draft PR
    ./scripts/create-pr.sh -b "wip/experimental" -t "WIP: Experimental feature" --draft

${YELLOW}QUALITY CHECKS:${NC}
    ${CHECK} Code formatting (gofmt)
    ${CHECK} Linting (go vet)
    ${CHECK} Tests pass
    ${CHECK} Version bump validation
    ${CHECK} Git status clean
    ${CHECK} Branch up to date

${YELLOW}VERSION REQUIREMENTS:${NC}
    This script will check if you've bumped the version in the VERSION file.
    If not, it will prompt you to do so using:
    
    make version-bump-patch    # For bug fixes
    make version-bump-minor    # For new features
    make version-bump-major    # For breaking changes

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
    if [[ -z "$BRANCH" ]]; then
        print_error "Branch name is required. Use -b or --branch"
        exit 1
    fi
    
    if [[ -z "$TITLE" ]]; then
        print_error "PR title is required. Use -t or --title"
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
        if [[ "$FORCE" == "false" ]]; then
            print_info "Commit your changes first, or use --force to continue"
            exit 1
        fi
    fi
    
    # Check current branch
    CURRENT_BRANCH=$(git branch --show-current)
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
    if ! git ls-remote --heads origin "$CURRENT_BRANCH" | grep -q "$CURRENT_BRANCH"; then
        print_info "Branch doesn't exist on remote, will push after checks"
        NEED_PUSH=true
    else
        print_info "Branch exists on remote"
        NEED_PUSH=false
        
        # Check if local is ahead of remote
        LOCAL=$(git rev-parse HEAD)
        REMOTE=$(git rev-parse "origin/$CURRENT_BRANCH" 2>/dev/null || echo "")
        
        if [[ "$LOCAL" != "$REMOTE" && -n "$REMOTE" ]]; then
            print_info "Local branch is ahead of remote, will push updates"
            NEED_PUSH=true
        fi
    fi
}

# Format code
format_code() {
    print_step "Checking code formatting..."
    
    # Check if code needs formatting
    UNFORMATTED=$(gofmt -l . 2>/dev/null || true)
    
    if [[ -n "$UNFORMATTED" ]]; then
        print_warning "Code needs formatting:"
        echo "$UNFORMATTED"
        
        if [[ "$AUTO_FIX" == "true" ]]; then
            print_step "Auto-fixing formatting..."
            gofmt -w .
            print_success "Code formatted successfully"
            
            # Add formatted files to git
            if [[ -n $(git status --porcelain) ]]; then
                git add .
                git commit -m "style: auto-format code with gofmt"
                print_success "Committed formatting changes"
            fi
        else
            print_error "Please format your code with: make fmt"
            exit 1
        fi
    else
        print_success "Code is properly formatted"
    fi
}

# Run linting
run_linting() {
    print_step "Running linting checks..."
    
    # Run go vet
    if ! go vet ./...; then
        print_error "go vet failed"
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
    
    if ! go test ./...; then
        print_error "Tests failed"
        print_info "Fix the failing tests before creating PR"
        exit 1
    fi
    
    print_success "All tests passed"
}

# Check version bump
check_version_bump() {
    print_step "Checking version bump..."
    
    if [[ ! -f "VERSION" ]]; then
        print_error "VERSION file not found"
        print_info "Create one with: echo '0.1.0' > VERSION"
        exit 1
    fi
    
    CURRENT_VERSION=$(cat VERSION | tr -d '\n' | tr -d ' ')
    
    # Get the version from main branch
    MAIN_VERSION=""
    if git show main:VERSION &>/dev/null; then
        MAIN_VERSION=$(git show main:VERSION | tr -d '\n' | tr -d ' ')
    fi
    
    if [[ -n "$MAIN_VERSION" && "$CURRENT_VERSION" == "$MAIN_VERSION" ]]; then
        print_warning "Version hasn't been bumped from main ($MAIN_VERSION)"
        print_info "You need to bump the version. Choose one:"
        echo ""
        echo -e "  ${GREEN}make version-bump-patch${NC}  # For bug fixes (x.y.Z)"
        echo -e "  ${GREEN}make version-bump-minor${NC}  # For new features (x.Y.z)"
        echo -e "  ${GREEN}make version-bump-major${NC}  # For breaking changes (X.y.z)"
        echo ""
        
        if [[ "$FORCE" == "false" ]]; then
            read -p "Would you like to bump the version now? [patch/minor/major/skip]: " choice
            case $choice in
                patch|p)
                    make version-bump-patch
                    print_success "Version bumped to patch level"
                    ;;
                minor|m)
                    make version-bump-minor
                    print_success "Version bumped to minor level"
                    ;;
                major|M)
                    make version-bump-major
                    print_success "Version bumped to major level"
                    ;;
                skip|s)
                    print_warning "Skipping version bump (PR may fail CI)"
                    ;;
                *)
                    print_error "Invalid choice. Exiting."
                    exit 1
                    ;;
            esac
        else
            print_warning "Skipping version bump due to --force flag"
        fi
    else
        print_success "Version bumped to $CURRENT_VERSION"
    fi
}

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
- [x] Version bumped (if needed)
- [x] Documentation updated (if needed)"
    fi
    
    # Create PR command
    PR_CMD="gh pr create --title \"$TITLE\" --body \"$PR_BODY\" --base main --head $CURRENT_BRANCH"
    
    if [[ "$DRAFT" == "true" ]]; then
        PR_CMD="$PR_CMD --draft"
    fi
    
    # Execute PR creation
    if eval "$PR_CMD"; then
        print_success "Pull request created successfully!"
        
        # Get PR URL
        PR_URL=$(gh pr view --json url --jq .url)
        print_info "PR URL: $PR_URL"
        
        # Open PR in browser (optional)
        read -p "Open PR in browser? [y/N]: " open_browser
        if [[ "$open_browser" =~ ^[Yy]$ ]]; then
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
    echo -e "  Branch: $CURRENT_BRANCH → main"
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
    validate_args
    
    # Pre-flight checks
    check_git_repo
    check_gh_cli
    check_gh_auth
    check_git_status
    check_and_push_branch
    
    # Code quality checks
    format_code
    run_linting
    run_tests
    check_version_bump
    build_project
    
    # Push and create PR
    push_branch
    create_pr
    
    # Summary
    show_summary
    
    print_success "All done! Your PR is ready for review."
}

# Run main function with all arguments
main "$@" 