# CI/CD Artifact Fetching

Integrate gdl into your CI/CD pipelines for reliable artifact downloads with automatic retry and resume capabilities.

## Use Case

- Download build dependencies in CI pipelines
- Fetch pre-built binaries for testing
- Pull release artifacts for deployment
- Download test fixtures and datasets

## Quick Start

```bash
# Install gdl in CI environment
go install github.com/forest6511/gdl/cmd/gdl@latest

# Download artifact with retry
gdl --retry 3 --quiet https://artifacts.example.com/build-123/app.tar.gz
```

## CI Platform Examples

### GitHub Actions

```yaml
name: Build and Deploy
on: [push]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install gdl
        run: |
          go install github.com/forest6511/gdl/cmd/gdl@latest
          gdl --version

      - name: Download Dependencies
        run: |
          gdl --retry 3 --quiet \
            -o deps/library.tar.gz \
            https://releases.example.com/library-v2.0.tar.gz

          gdl --retry 3 --quiet \
            -H "Authorization: Bearer ${{ secrets.ARTIFACT_TOKEN }}" \
            -o deps/private-dep.zip \
            https://private-artifacts.example.com/dep.zip

      - name: Build
        run: |
          tar -xzf deps/library.tar.gz -C vendor/
          make build
```

### GitLab CI

```yaml
stages:
  - download
  - build

variables:
  ARTIFACT_URL: "https://artifacts.example.com"

download-deps:
  stage: download
  image: golang:1.22
  before_script:
    - go install github.com/forest6511/gdl/cmd/gdl@latest
  script:
    - gdl --retry 3 --quiet -o vendor.tar.gz ${ARTIFACT_URL}/vendor-cache.tar.gz
    - tar -xzf vendor.tar.gz
  artifacts:
    paths:
      - vendor/
    expire_in: 1 hour

build:
  stage: build
  image: golang:1.22
  needs: [download-deps]
  script:
    - make build
```

### Jenkins Pipeline

```groovy
pipeline {
    agent any

    stages {
        stage('Setup') {
            steps {
                sh '''
                    go install github.com/forest6511/gdl/cmd/gdl@latest
                '''
            }
        }

        stage('Download Artifacts') {
            steps {
                withCredentials([string(credentialsId: 'artifact-token', variable: 'TOKEN')]) {
                    sh '''
                        gdl --retry 3 --quiet \
                            -H "Authorization: Bearer ${TOKEN}" \
                            -o artifacts/app.tar.gz \
                            https://artifacts.example.com/latest/app.tar.gz
                    '''
                }
            }
        }

        stage('Build') {
            steps {
                sh '''
                    tar -xzf artifacts/app.tar.gz
                    make build
                '''
            }
        }
    }
}
```

### CircleCI

```yaml
version: 2.1

jobs:
  build:
    docker:
      - image: cimg/go:1.22
    steps:
      - checkout

      - run:
          name: Install gdl
          command: |
            go install github.com/forest6511/gdl/cmd/gdl@latest

      - run:
          name: Download Dependencies
          command: |
            gdl --retry 3 --quiet -o /tmp/deps.tar.gz \
              https://deps.example.com/go-deps-v1.tar.gz
            tar -xzf /tmp/deps.tar.gz -C vendor/

      - run:
          name: Build
          command: make build

workflows:
  build-and-test:
    jobs:
      - build
```

## Best Practices for CI

### 1. Always Use Retry

CI environments can have network issues. Always enable retry:

```bash
gdl --retry 5 --quiet URL
```

### 2. Use Quiet Mode

Reduce log noise in CI pipelines:

```bash
gdl --quiet URL  # Minimal output
gdl --progress simple URL  # Simple progress line
```

### 3. Set Appropriate Timeouts

Prevent hanging builds:

```bash
gdl --timeout 300s --retry 3 URL  # 5 minute timeout
```

### 4. Cache Downloads

Use CI caching to avoid re-downloading:

```yaml
# GitHub Actions example
- uses: actions/cache@v3
  with:
    path: ~/.gdl-cache
    key: deps-${{ hashFiles('dependencies.lock') }}

- run: |
    if [ ! -f ~/.gdl-cache/deps.tar.gz ]; then
      gdl -o ~/.gdl-cache/deps.tar.gz https://deps.example.com/deps.tar.gz
    fi
    cp ~/.gdl-cache/deps.tar.gz ./deps.tar.gz
```

### 5. Handle Authenticated Downloads

```bash
# Using header
gdl -H "Authorization: Bearer ${TOKEN}" URL

# Using environment variable for proxy
HTTP_PROXY=http://proxy:8080 gdl URL
```

## Error Handling in Scripts

```bash
#!/bin/bash
set -e

# Download with error handling
if ! gdl --retry 3 --quiet -o artifact.tar.gz "$ARTIFACT_URL"; then
    echo "Failed to download artifact after retries"
    exit 1
fi

# Verify download
if [ ! -f artifact.tar.gz ]; then
    echo "Artifact file not found"
    exit 1
fi

# Check file size (example: must be > 1KB)
SIZE=$(stat -f%z artifact.tar.gz 2>/dev/null || stat --printf="%s" artifact.tar.gz)
if [ "$SIZE" -lt 1024 ]; then
    echo "Downloaded file too small, may be corrupted"
    exit 1
fi

echo "Artifact downloaded successfully"
```

## Performance in CI

### Parallel Downloads

Download multiple artifacts in parallel:

```bash
#!/bin/bash
# Download multiple files in parallel
gdl --quiet -o dep1.tar.gz https://deps.example.com/dep1.tar.gz &
gdl --quiet -o dep2.tar.gz https://deps.example.com/dep2.tar.gz &
gdl --quiet -o dep3.tar.gz https://deps.example.com/dep3.tar.gz &
wait
echo "All downloads completed"
```

### Bandwidth Management

Share bandwidth with other CI jobs:

```bash
gdl --max-rate 10MB/s --quiet URL
```

## Related

- [CDN Large File Delivery](cdn-delivery.md) - Optimize large downloads
- [Batch Downloads](batch-downloads.md) - Multiple file strategies
