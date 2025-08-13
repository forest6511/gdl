#!/bin/bash
# Update CHANGELOG.md for new release or prepare full release

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse arguments
FULL_RELEASE=false
VERSION=${1:-}

# Check for --release flag
if [ "$1" = "--release" ]; then
    FULL_RELEASE=true
    VERSION=${2:-}
fi

if [ -z "$VERSION" ]; then
    echo -e "${RED}Usage: $0 [--release] <version>${NC}"
    echo "Examples:"
    echo "  $0 v0.10.0                    # Update CHANGELOG only"
    echo "  $0 --release v0.10.0          # Full release preparation"
    exit 1
fi

# Validate version format
if ! [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${RED}Error: Version must be in format vX.Y.Z (e.g., v0.10.0)${NC}"
    exit 1
fi

DATE=$(date +%Y-%m-%d)
VERSION_NUM=${VERSION#v}

echo -e "${BLUE}Processing release ${VERSION}${NC}"

# If full release, check branch and run pre-release checks
if [ "$FULL_RELEASE" = true ]; then
    # Check if on main branch
    BRANCH=$(git rev-parse --abbrev-ref HEAD)
    if [ "$BRANCH" != "main" ]; then
        echo -e "${YELLOW}Warning: Not on main branch (currently on: $BRANCH)${NC}"
        read -p "Continue? (y/n) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo -e "${RED}Aborted${NC}"
            exit 1
        fi
    fi

    # Check for uncommitted changes
    if ! git diff-index --quiet HEAD --; then
        echo -e "${RED}Error: You have uncommitted changes${NC}"
        echo "Please commit or stash your changes before creating a release"
        exit 1
    fi

    # Run tests
    echo -e "${GREEN}Running tests...${NC}"
    if ! go test ./...; then
        echo -e "${RED}Tests failed! Cannot proceed with release${NC}"
        exit 1
    fi

    echo -e "${GREEN}✓ Pre-release checks passed${NC}"
fi

# Check if CHANGELOG.md exists
if [ ! -f "CHANGELOG.md" ]; then
    echo -e "${RED}Error: CHANGELOG.md not found${NC}"
    exit 1
fi

# Check if version already exists
if grep -q "## \[$VERSION_NUM\]" CHANGELOG.md; then
    echo -e "${RED}Error: Version $VERSION_NUM already exists in CHANGELOG.md${NC}"
    exit 1
fi

# Create temporary file with new version section
cat > /tmp/new_version.md << EOF
## [$VERSION_NUM] - $DATE

### Added
- 

### Changed
- 

### Fixed
- 

### Security
- 

EOF

# Insert new version section after [Unreleased]
awk '/## \[Unreleased\]/{print; print ""; system("cat /tmp/new_version.md"); next}1' CHANGELOG.md > CHANGELOG.tmp
mv CHANGELOG.tmp CHANGELOG.md

# Update comparison links
PREV_VERSION=$(grep -oE '\[[0-9]+\.[0-9]+\.[0-9]+\]' CHANGELOG.md | head -2 | tail -1 | tr -d '[]')

# Update Unreleased link
sed -i.bak "s|\[Unreleased\]:.*|[Unreleased]: https://github.com/forest6511/godl/compare/${VERSION}...HEAD|" CHANGELOG.md

# Add new comparison link before the last line
if [ -n "$PREV_VERSION" ]; then
    echo "[${VERSION_NUM}]: https://github.com/forest6511/godl/compare/v${PREV_VERSION}...${VERSION}" >> CHANGELOG.md
else
    echo "[${VERSION_NUM}]: https://github.com/forest6511/godl/releases/tag/${VERSION}" >> CHANGELOG.md
fi

# Clean up backup file
rm -f CHANGELOG.md.bak

echo -e "${GREEN}✓ CHANGELOG.md updated for version $VERSION${NC}"

# If full release, commit and tag
if [ "$FULL_RELEASE" = true ]; then
    echo -e "${GREEN}Committing CHANGELOG changes...${NC}"
    git add CHANGELOG.md
    git commit -m "chore: prepare release ${VERSION}

Update CHANGELOG.md with version ${VERSION} section"

    echo -e "${GREEN}Creating git tag...${NC}"
    git tag -a $VERSION -m "Release ${VERSION}

See CHANGELOG.md for details."

    echo -e "${GREEN}Pushing to remote...${NC}"
    git push origin main
    git push origin $VERSION

    echo -e "${GREEN}🎉 Release ${VERSION} prepared successfully!${NC}"
    echo -e "${BLUE}GitHub Actions will now build and publish the release.${NC}"
    echo -e "${YELLOW}Next steps:${NC}"
    echo "1. Check the GitHub Actions status"
    echo "2. Edit the GitHub release notes if needed"
    echo "3. Announce the release"
else
    echo -e "${YELLOW}! Please edit CHANGELOG.md to add release notes${NC}"
    echo -e "${BLUE}Then run: $0 --release $VERSION${NC}"
fi