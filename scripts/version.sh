#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
VERSION_FILE="VERSION"
CHANGELOG_FILE="CHANGELOG.md"

# Function to print colored output
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to get current version
get_current_version() {
    if [[ -f "$VERSION_FILE" ]]; then
        cat "$VERSION_FILE"
    else
        echo "0.0.0"
    fi
}

# Function to validate semver format
validate_version() {
    local version=$1
    if [[ ! $version =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9\.-]+)?$ ]]; then
        print_error "Invalid version format: $version"
        print_error "Expected format: X.Y.Z or X.Y.Z-prerelease"
        exit 1
    fi
}

# Function to compare versions
version_gt() {
    test "$(printf '%s\n' "$@" | sort -V | head -n 1)" != "$1"
}

# Function to bump version
bump_version() {
    local current_version="$1"
    local bump_type="$2"
    local prerelease_id="$3"
    
    # Parse current version
    if [[ $current_version =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)(-(.+))?$ ]]; then
        local major="${BASH_REMATCH[1]}"
        local minor="${BASH_REMATCH[2]}"
        local patch="${BASH_REMATCH[3]}"
        local prerelease="${BASH_REMATCH[5]}"
    else
        print_error "Invalid current version format: $current_version"
        exit 1
    fi
    
    case $bump_type in
        "major")
            major=$((major + 1))
            minor=0
            patch=0
            prerelease=""
            ;;
        "minor")
            minor=$((minor + 1))
            patch=0
            prerelease=""
            ;;
        "patch")
            patch=$((patch + 1))
            prerelease=""
            ;;
        "prerelease")
            if [[ -z "$prerelease_id" ]]; then
                prerelease_id="alpha"
            fi
            
            if [[ -n "$prerelease" ]]; then
                # If already a prerelease, increment the number
                if [[ $prerelease =~ ^(.+)\.([0-9]+)$ ]]; then
                    local pre_id="${BASH_REMATCH[1]}"
                    local pre_num="${BASH_REMATCH[2]}"
                    if [[ "$pre_id" == "$prerelease_id" ]]; then
                        prerelease="$pre_id.$((pre_num + 1))"
                    else
                        prerelease="$prerelease_id.1"
                    fi
                else
                    prerelease="$prerelease.1"
                fi
            else
                # First prerelease
                patch=$((patch + 1))
                prerelease="$prerelease_id.1"
            fi
            ;;
        *)
            print_error "Invalid bump type: $bump_type"
            print_error "Valid types: major, minor, patch, prerelease"
            exit 1
            ;;
    esac
    
    if [[ -n "$prerelease" ]]; then
        echo "${major}.${minor}.${patch}-${prerelease}"
    else
        echo "${major}.${minor}.${patch}"
    fi
}

# Function to update version file
update_version_file() {
    local new_version="$1"
    echo "$new_version" > "$VERSION_FILE"
    print_success "Updated $VERSION_FILE to $new_version"
}

# Function to create/update changelog
update_changelog() {
    local version="$1"
    local date=$(date +"%Y-%m-%d")
    
    if [[ ! -f "$CHANGELOG_FILE" ]]; then
        cat > "$CHANGELOG_FILE" << EOF
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [$version] - $date

### Added
- Initial release

EOF
    else
        # Insert new version entry after [Unreleased]
        local temp_file=$(mktemp)
        awk -v version="$version" -v date="$date" '
        /^## \[Unreleased\]/ {
            print $0
            print ""
            print "## [" version "] - " date
            print ""
            print "### Added"
            print "- "
            print ""
            print "### Changed"
            print "- "
            print ""
            print "### Fixed"
            print "- "
            print ""
            next
        }
        { print }
        ' "$CHANGELOG_FILE" > "$temp_file"
        
        mv "$temp_file" "$CHANGELOG_FILE"
    fi
    
    print_success "Updated $CHANGELOG_FILE"
}

# Function to check if there are only version-related changes
has_only_version_changes() {
    # Get list of changed files that are not VERSION or CHANGELOG.md
    local other_changes=$(git status --porcelain | grep -v "$VERSION_FILE" | grep -v "$CHANGELOG_FILE")
    
    # If there are no other changes, return true (0)
    if [[ -z "$other_changes" ]]; then
        return 0
    else
        return 1
    fi
}

# Function to commit and tag
commit_and_tag() {
    local version="$1"
    local tag="v$version"
    
    # Check if we're in a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        print_error "Not in a git repository"
        exit 1
    fi
    
    # Get current branch
    local current_branch=$(git rev-parse --abbrev-ref HEAD)
    
    # Check if working directory is clean
    if [[ -n $(git status --porcelain) ]]; then
        print_warning "Working directory is not clean. Staging version files..."
        git add "$VERSION_FILE" "$CHANGELOG_FILE"
    fi
    
    # Commit changes
    git commit -m "chore: bump version to $version" 2>/dev/null || true
    
    # Create tag
    if git tag -l | grep -q "^$tag$"; then
        print_warning "Tag $tag already exists"
    else
        git tag -a "$tag" -m "Release $version"
        print_success "Created tag $tag"
    fi
    
    # Auto-push if there are only version-related changes
    if has_only_version_changes; then
        print_info "Only version-related changes detected. Auto-pushing..."
        git push origin "$current_branch"
        git push origin "$tag"
        print_success "Changes pushed successfully!"
    else
        print_warning "Other changes detected besides version files."
        print_info "To push changes and tags, run:"
        print_info "  git push origin $current_branch && git push origin $tag"
    fi
}

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 <command> [options]

Commands:
    current                 Show current version
    bump <type> [id]        Bump version (major|minor|patch|prerelease [id])
    set <version>           Set specific version
    tag                     Create git tag for current version
    release <type> [id]     Bump version, update changelog, commit and tag

Examples:
    $0 current              # Show current version
    $0 bump patch           # Bump patch version (1.0.0 -> 1.0.1)
    $0 bump minor           # Bump minor version (1.0.1 -> 1.1.0)
    $0 bump major           # Bump major version (1.1.0 -> 2.0.0)
    $0 bump prerelease      # Bump prerelease (1.0.0 -> 1.0.1-alpha.1)
    $0 bump prerelease beta # Bump prerelease with custom id (1.0.0 -> 1.0.1-beta.1)
    $0 set 2.1.0            # Set specific version
    $0 release patch        # Full release: bump, changelog, commit, tag
EOF
}

# Main script logic
case "${1:-}" in
    "current")
        echo "$(get_current_version)"
        ;;
    "bump")
        if [[ -z "${2:-}" ]]; then
            print_error "Missing bump type"
            show_usage
            exit 1
        fi
        
        current=$(get_current_version)
        new_version=$(bump_version "$current" "$2" "${3:-}")
        
        print_info "Current version: $current"
        print_info "New version: $new_version"
        
        update_version_file "$new_version"
        ;;
    "set")
        if [[ -z "${2:-}" ]]; then
            print_error "Missing version"
            show_usage
            exit 1
        fi
        
        validate_version "$2"
        current=$(get_current_version)
        
        if ! version_gt "$2" "$current"; then
            print_error "New version ($2) must be greater than current version ($current)"
            exit 1
        fi
        
        update_version_file "$2"
        ;;
    "tag")
        current=$(get_current_version)
        commit_and_tag "$current"
        ;;
    "release")
        if [[ -z "${2:-}" ]]; then
            print_error "Missing release type"
            show_usage
            exit 1
        fi
        
        current=$(get_current_version)
        new_version=$(bump_version "$current" "$2" "${3:-}")
        
        print_info "Creating release $new_version"
        print_info "Current version: $current"
        print_info "New version: $new_version"
        
        update_version_file "$new_version"
        update_changelog "$new_version"
        commit_and_tag "$new_version"
        
        print_success "Release $new_version created successfully!"
        ;;
    *)
        show_usage
        exit 1
        ;;
esac 