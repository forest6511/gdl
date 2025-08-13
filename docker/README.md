# Docker

Docker configuration for godl.

## Files

- `Dockerfile`: Multi-stage build for minimal image (~10MB)
- `.dockerignore`: Excludes unnecessary files from build context

## Usage

### Build locally
```bash
docker build -f docker/Dockerfile -t godl:local .
```

### Run
```bash
# Show help
docker run --rm godl:local --help

# Download a file to current directory
docker run --rm -v $(pwd):/downloads godl:local \
  -o /downloads/file.zip \
  https://example.com/file.zip
```

### Available images (when published via GoReleaser)
- `ghcr.io/forest6511/godl:latest`
- `ghcr.io/forest6511/godl:v0.9.0`

## Image details

- **Base**: scratch (minimal)
- **Size**: ~10MB
- **Security**: CA certificates included for HTTPS
- **Architecture**: Built for linux/amd64