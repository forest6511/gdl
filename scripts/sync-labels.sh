#!/bin/bash
# Sync labels to GitHub

set -e

REPO="forest6511/gdl"
LABELS_FILE=".github/labels.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if labels file exists
if [ ! -f "$LABELS_FILE" ]; then
    echo -e "${RED}Error: $LABELS_FILE not found${NC}"
    exit 1
fi

# Check for required tools
if ! command -v gh &> /dev/null; then
    echo -e "${RED}GitHub CLI not found. Please install: https://cli.github.com/${NC}"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    echo -e "${RED}jq not found. Please install jq${NC}"
    exit 1
fi

# Check GitHub CLI authentication
if ! gh auth status &> /dev/null; then
    echo -e "${RED}Not authenticated with GitHub CLI. Please run: gh auth login${NC}"
    exit 1
fi

echo -e "${YELLOW}Starting label sync for $REPO...${NC}"

# Delete existing labels (with confirmation)
echo -e "${YELLOW}Deleting existing labels...${NC}"
existing_labels=$(gh label list -R $REPO --json name -q '.[].name' 2>/dev/null || echo "")
if [ -n "$existing_labels" ]; then
    echo "$existing_labels" | while read -r label; do
        if [ -n "$label" ]; then
            gh label delete "$label" -R $REPO --yes 2>/dev/null && \
                echo -e "  ${RED}✗${NC} Deleted: $label"
        fi
    done
else
    echo -e "  No existing labels found"
fi

# Create new labels from JSON
echo -e "${YELLOW}Creating new labels...${NC}"
cat "$LABELS_FILE" | jq -r '.[] | "\(.name)|\(.color)|\(.description)"' | while IFS='|' read -r name color description; do
    if gh label create "$name" -R $REPO --color "$color" --description "$description" 2>/dev/null; then
        echo -e "  ${GREEN}✓${NC} Created: $name"
    else
        # If creation fails, try updating existing label
        if gh label edit "$name" -R $REPO --color "$color" --description "$description" 2>/dev/null; then
            echo -e "  ${GREEN}✓${NC} Updated: $name"
        else
            echo -e "  ${RED}✗${NC} Failed: $name"
        fi
    fi
done

echo -e "${GREEN}Labels synced successfully!${NC}"