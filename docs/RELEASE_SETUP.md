# Release Setup Guide

## GitHub Secrets Configuration

To enable automated releases with GoReleaser, configure the following GitHub Secrets:

### Required Secrets

1. **GITHUB_TOKEN** (Automatically available)
   - Used for creating releases and uploading assets
   - No configuration needed

### Optional Secrets (for advanced features)

2. **HOMEBREW_TAP_GITHUB_TOKEN**
   - Purpose: Auto-update Homebrew formula
   - Setup:
     1. Create a Personal Access Token at https://github.com/settings/tokens
     2. Select scope: `repo` (full control of private repositories)
     3. Add to repository secrets: Settings → Secrets → Actions → New repository secret
     4. Name: `HOMEBREW_TAP_GITHUB_TOKEN`
     5. Value: Your personal access token

3. **SCOOP_GITHUB_TOKEN**
   - Purpose: Auto-update Scoop manifest (Windows)
   - Setup: Same as HOMEBREW_TAP_GITHUB_TOKEN
   - Only needed if you create a scoop-bucket repository

## Docker Image Publishing

GitHub Container Registry (ghcr.io) uses GITHUB_TOKEN automatically.
No additional configuration needed.

## Release Workflows

### Option 1: Automated Release (Recommended)
```bash
# Step 1: Update CHANGELOG only
./scripts/prepare-release.sh v0.10.0
# Edit CHANGELOG.md to add release notes

# Step 2: Full release preparation and push
./scripts/prepare-release.sh --release v0.10.0
```
Uses: `.github/workflows/release.yml`

### Option 2: Manual Release
```bash
# Create and push tag manually
git tag v0.10.0
git push origin v0.10.0
```

### Option 3: GoReleaser (Advanced)
Uses: `.github/workflows/release-goreleaser.yml`

Features:
- Docker images
- Homebrew formula updates
- Linux packages (deb/rpm/apk)
- SBOM generation

Same tag process as Option 1/2

## Testing Docker Image Locally

```bash
# Build
docker build -f docker/Dockerfile -t gdl:local .

# Run
docker run --rm gdl:local --help

# Download a file
docker run --rm -v $(pwd):/downloads gdl:local \
  -o /downloads/file.zip \
  https://example.com/file.zip
```

## Homebrew Formula Updates

### Manual Update
```bash
# After release, run:
./scripts/update-homebrew.sh v0.10.0
```

### Automatic Update (with GoReleaser)
Requires HOMEBREW_TAP_GITHUB_TOKEN secret.

## Checklist Before Release

### Using Automated Script (Recommended)
- [ ] Run: `./scripts/prepare-release.sh v0.X.Y`
- [ ] Edit CHANGELOG.md with release notes
- [ ] Run: `./scripts/prepare-release.sh --release v0.X.Y`
- [ ] Verify release on GitHub
- [ ] Test installation methods

### Manual Process
- [ ] Update CHANGELOG.md with release notes
- [ ] Run tests: `go test ./...`
- [ ] Run linter: `golangci-lint run`
- [ ] Update version in documentation if needed
- [ ] Create and push tag
- [ ] Verify release on GitHub
- [ ] Test installation methods:
  - [ ] Direct binary download
  - [ ] `go install`
  - [ ] `brew install` (after formula update)
  - [ ] Docker image (if using GoReleaser)