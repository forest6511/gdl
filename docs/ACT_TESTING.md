# Act - Local GitHub Actions Testing

[Act](https://github.com/nektos/act) allows you to run GitHub Actions locally for testing.

## Installation

```bash
# macOS
brew install act

# Other platforms
# See: https://github.com/nektos/act#installation
```

## Configuration

The project includes `.actrc` for default settings:
- Container architecture: linux/amd64
- Artifact server path: /tmp/artifacts

## Usage

### List available workflows
```bash
act --list
```

### Run specific workflows

#### Quick checks (fast)
```bash
act push -j quick-checks
```

#### Full CI pipeline (slow)
```bash
act push -W .github/workflows/main.yml
```

#### Release workflow (dry run)
```bash
act push -W .github/workflows/release.yml --dryrun
```

### Run with specific events
```bash
# Test pull request workflows
act pull_request

# Test specific workflow file
act -W .github/workflows/lint.yml
```

## Secrets

Copy `.github/act_secrets` to `.env` and add your tokens:

```bash
cp .github/act_secrets .env
# Edit .env with your tokens
```

Then run with secrets:
```bash
act --env-file .env
```

## Known Issues

### ARM64 Macs
- Use `--container-architecture linux/amd64` (already in .actrc)
- Some actions may have Node.js path issues (harmless for most testing)

### Large Workflows
- Use `--dryrun` first to validate
- Use `-j job-name` to run specific jobs
- Some jobs may timeout (increase with `--timeout`)

## Useful Commands

```bash
# Dry run to validate workflow syntax
act --dryrun

# Run with verbose output
act -v

# Run specific job
act -j test

# List all jobs in workflow
act --list -W .github/workflows/main.yml

# Clean up containers after run
act --rm
```

## Workflow-Specific Notes

### Main CI (`main.yml`)
- Quick checks: Fast validation
- Full pipeline: ~5-10 minutes locally
- Use `-j quick-checks` for rapid feedback

### Release (`release.yml`)
- Use `--dryrun` to validate without creating releases
- Requires proper version tags for full testing
- Test with: `act push --eventpath .github/test-events/release.json`

### Security (`security.yml`)
- May require specific tokens
- Some scans may not work in local containers

## Performance Tips

1. **Use job filters**: `-j job-name` for specific jobs
2. **Dry run first**: `--dryrun` to catch syntax errors
3. **Skip slow jobs**: Focus on critical paths
4. **Cache Docker images**: Act reuses pulled images
5. **Use smaller images**: Consider using `node:alpine` variants

## Troubleshooting

### Container Issues
```bash
# Force pull latest images
act --pull

# Clean up and restart
docker system prune
act --rm
```

### Permission Issues
```bash
# Check Docker permissions
docker ps

# Ensure Docker daemon is running
brew services restart docker
```