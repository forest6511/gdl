#!/bin/bash
# Setup git hooks for commit message validation

set -e

# Check if we're in a git repository
if [ ! -d ".git" ]; then
    echo "Error: Not in a git repository"
    exit 1
fi

# Create hooks directory if it doesn't exist
mkdir -p .git/hooks

echo "Installing commit message validation hook..."

cat > .git/hooks/commit-msg << 'EOF'
#!/bin/bash
# Validate commit message format

commit_regex='^(feat|fix|docs|style|refactor|perf|test|chore)(\([[:alnum:]._-]+\))?: .{1,50}'

if ! grep -qE "$commit_regex" "$1"; then
    echo "❌ Invalid commit message format!"
    echo ""
    echo "📝 Expected format: <type>(<scope>): <subject>"
    echo "📝 Example: feat(cli): add progress bar support"
    echo ""
    echo "🏷️  Valid types:"
    echo "   feat     - New feature"
    echo "   fix      - Bug fix"  
    echo "   docs     - Documentation changes"
    echo "   style    - Code style changes (formatting, etc.)"
    echo "   refactor - Code refactoring"
    echo "   perf     - Performance improvements"
    echo "   test     - Adding or updating tests"
    echo "   chore    - Maintenance tasks"
    echo ""
    echo "🎯 Valid scopes: cli, api, core, plugin, docs"
    echo ""
    exit 1
fi

# Check subject length
subject=$(head -n1 "$1" | sed 's/^[^:]*: //')
if [ ${#subject} -gt 50 ]; then
    echo "⚠️  Warning: Subject line is too long (${#subject} chars, max 50)"
    echo "Consider making it more concise"
fi

echo "✅ Commit message format is valid"
EOF

chmod +x .git/hooks/commit-msg

# Also create a pre-commit hook for additional checks
echo "Installing pre-commit validation hook..."

cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
# Pre-commit checks

set -e

# Check if Go files are properly formatted
if command -v gofmt >/dev/null 2>&1; then
    unformatted=$(gofmt -l . | grep -v vendor/ || true)
    if [ -n "$unformatted" ]; then
        echo "❌ Go files are not properly formatted:"
        echo "$unformatted"
        echo ""
        echo "Run: go fmt ./..."
        exit 1
    fi
fi

# Run tests if they exist
if [ -f "go.mod" ] && command -v go >/dev/null 2>&1; then
    echo "🧪 Running tests..."
    if ! go test ./... >/dev/null 2>&1; then
        echo "❌ Tests failed. Please fix before committing."
        echo "Run: go test ./... for details"
        exit 1
    fi
    echo "✅ Tests passed"
fi

echo "✅ Pre-commit checks passed"
EOF

chmod +x .git/hooks/pre-commit

echo ""
echo "🎉 Git hooks installed successfully!"
echo ""
echo "📋 Installed hooks:"
echo "   • commit-msg: Validates conventional commit format"
echo "   • pre-commit: Runs formatting and test checks"
echo ""
echo "💡 To set up commit message template:"
echo "   git config commit.template .gitmessage"