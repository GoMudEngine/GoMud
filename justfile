# DOGMud Development Justfile
# Quick commands for common development tasks
# Install just: https://github.com/casey/just

# Use sh for all commands. On Windows (PowerShell/CMD), use Git's sh.exe.
set shell := ["sh", "-cu"]
set windows-shell := ["C:/Program Files/Git/bin/sh.exe", "-cu"]

# List all available commands
default:
    @just --list

# Build the DOGMud server
build:
    @echo "Building DOGMud server..."
    go generate
    go build -trimpath -a -o go-mud-server

# Run the DOGMud server
run:
    @echo "Starting DOGMud server..."
    go generate
    go run .

# Run with fresh instance data (clean slate)
run-fresh:
    @echo "Starting DOGMud with fresh instance data..."
    rm -rf _datafiles/world/dogmud/rooms.instances
    go generate
    go run .

# Run all tests
test:
    @echo "Running all tests..."
    go test ./...

# Run tests with verbose output
test-verbose:
    @echo "Running tests (verbose)..."
    go test -v ./...

# Run tests with coverage report
test-coverage:
    @echo "Running tests with coverage..."
    mkdir -p bin/covdatafiles
    go test ./... -coverprofile=bin/covdatafiles/cover.out
    go tool cover -html=bin/covdatafiles/cover.out

# Run only specific package tests (usage: just test-package internal/characters)
test-package PKG:
    @echo "Running tests for {{PKG}}..."
    go test -v ./{{PKG}}

# Format all Go code
fmt:
    @echo "Formatting Go code..."
    go fmt ./...

# Run Go vet for static analysis
vet:
    @echo "Running go vet..."
    go vet ./...

# Validate code (format check + vet)
validate:
    @echo "Validating code..."
    go vet ./...

# Clean build artifacts and Docker containers
clean:
    @echo "Cleaning build artifacts..."
    make clean

# Clean room instance data (fresh world)
clean-instances:
    @echo "Cleaning room instances..."
    rm -rf _datafiles/world/dogmud/rooms.instances
    @echo "Room instances cleaned!"

# Clean user data (WARNING: Deletes all saved characters)
clean-users:
    @echo "WARNING: This will delete all user data!"
    @echo "Press Ctrl+C to cancel, or wait 3 seconds..."
    @sleep 3
    rm -rf _datafiles/world/dogmud/users/*
    @echo "User data cleaned!"

# Generate module imports
generate:
    @echo "Generating module imports..."
    go generate

# Build for Windows 64-bit
build-win64:
    @echo "Building for Windows 64-bit..."
    make build_win64

# Build for Linux 64-bit
build-linux64:
    @echo "Building for Linux 64-bit..."
    make build_linux64

# Build for Raspberry Pi Zero 2W
build-rpi:
    @echo "Building for Raspberry Pi Zero 2W..."
    make build_rpi_zero2w

# Run linter on JavaScript code
lint-js:
    @echo "Linting JavaScript..."
    make js-lint

# Quick development cycle: format, validate, build, test
dev:
    @echo "Running development cycle..."
    @just fmt
    @just validate
    @just build
    @just test

# Quick check before commit: validate and test
check:
    @echo "Running pre-commit checks..."
    @just validate
    @just test
    @echo "✓ All checks passed!"

# Manual test: Start server and show connection info
manual-test:
    @echo "╔════════════════════════════════════════╗"
    @echo "║       DOGMud Manual Test Session      ║"
    @echo "╚════════════════════════════════════════╝"
    @echo ""
    @echo "Telnet: localhost:33333"
    @echo "Web Client: http://localhost/webclient"
    @echo ""
    @echo "Default credentials:"
    @echo "  Username: admin"
    @echo "  Password: password"
    @echo ""
    @echo "Starting server..."
    @just run

# Show current git status
status:
    @git status

# Create a new feature branch (usage: just branch stage-1.1-rename-stats)
branch NAME:
    @echo "Creating feature branch: feature/{{NAME}}"
    git checkout development 2>/dev/null || git checkout -b development
    git checkout -b feature/{{NAME}}
    @echo "✓ Branch created: feature/{{NAME}}"

# Commit changes with conventional commit format (usage: just commit "feat: description")
commit MSG:
    @echo "Committing: {{MSG}}"
    git add .
    git commit -m "{{MSG}}\n\nCo-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"

# Merge feature branch to development (usage: just merge stage-1.1-rename-stats)
merge BRANCH:
    @echo "Merging feature/{{BRANCH}} to development..."
    git checkout development
    git merge --no-ff feature/{{BRANCH}} -m "merge: Merge feature/{{BRANCH}} into development"
    @echo "✓ Merged feature/{{BRANCH}} to development"

# Show git log with pretty formatting
log:
    git log --oneline --graph --decorate -10

# Show detailed log for last 5 commits
log-detail:
    git log -5 --pretty=format:"%h - %an, %ar : %s" --stat

# Stage 1.1: Rename stats (first dev stage)
stage-1-1:
    @echo "Starting Stage 1.1: Rename Stats"
    @just branch stage-1.1-rename-stats
    @echo "Ready to work on Stage 1.1!"

# Backup current world data
backup:
    @echo "Backing up DOGMud world data..."
    @mkdir -p backups
    @tar -czf backups/dogmud-world-$(date +%Y%m%d-%H%M%S).tar.gz _datafiles/world/dogmud
    @echo "✓ Backup created in backups/"

# Restore world data from backup (usage: just restore backups/dogmud-world-20260205-120000.tar.gz)
restore BACKUP:
    @echo "Restoring from {{BACKUP}}..."
    @tar -xzf {{BACKUP}} -C /
    @echo "✓ Restored from {{BACKUP}}"

# Check if server is running
check-server:
    @echo "Checking if DOGMud server is running..."
    @curl -s http://localhost/webclient > /dev/null && echo "✓ Server is running" || echo "✗ Server is not running"

# Kill running server process (if stuck)
kill-server:
    @echo "Looking for running DOGMud processes..."
    @pkill -f go-mud-server || echo "No running server found"

# Development workflow help
help-workflow:
    @echo "╔════════════════════════════════════════╗"
    @echo "║     DOGMud Development Workflow        ║"
    @echo "╚════════════════════════════════════════╝"
    @echo ""
    @echo "Starting a new stage:"
    @echo "  just branch stage-X.Y-description"
    @echo "  (make changes)"
    @echo "  just check"
    @echo "  just commit 'feat: description'"
    @echo "  just merge stage-X.Y-description"
    @echo ""
    @echo "Quick dev cycle:"
    @echo "  just dev          # Format, validate, build, test"
    @echo "  just check        # Quick pre-commit check"
    @echo "  just manual-test  # Start server for manual testing"
    @echo ""
    @echo "Common commands:"
    @echo "  just run          # Run server"
    @echo "  just run-fresh    # Run with clean instance data"
    @echo "  just test         # Run tests"
    @echo "  just clean        # Clean build artifacts"
    @echo ""
    @echo "Git commands:"
    @echo "  just status       # Git status"
    @echo "  just log          # Git log (pretty)"
    @echo "  just branch NAME  # Create feature branch"
    @echo "  just commit MSG   # Commit with message"
    @echo "  just merge BRANCH # Merge to development"
    @echo ""
    @echo "AI Testing:"
    @echo "  just ai-player       # Launch AI player"
    @echo "  just ai-player-fresh # Nuke AI save & relaunch"
    @echo ""
    @echo "Coverage:"
    @echo "  just help-coverage   # Check helpfile coverage"

# Launch the AI player bot (requires server running with AIPort enabled)
ai-player:
    @echo "Launching AI player..."
    @echo "  Ensure the MUD server is running with AIPort enabled (port 55555)"
    @echo ""
    python tools/ai_player.py

# Delete AI player's save file and launch a fresh AI player
ai-player-fresh:
    @echo "Nuking AI player save file..."
    @for f in _datafiles/world/dogmud/users/*.yaml; do \
        if grep -q "^username: aitester$$" "$$f" 2>/dev/null; then \
            echo "  Removing $$f"; \
            rm -f "$$f"; \
        fi; \
    done
    @echo "AI player save deleted. Launching fresh AI player..."
    @echo ""
    python tools/ai_player.py

# Check helpfile coverage (which commands lack help files)
help-coverage:
    @python tools/help_coverage.py
