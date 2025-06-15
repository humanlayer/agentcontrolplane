#!/bin/bash
# cleanup_coding_workers.sh - Cleans up a specific worker's worktree, tmux window, and kind cluster
# Usage: ./cleanup_coding_workers.sh <window_name>

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to log messages
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR:${NC} $1" >&2
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARN:${NC} $1"
}

info() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO:${NC} $1"
}

# Parse arguments
if [ $# -ne 1 ]; then
    echo "Usage: $0 <window_name>"
    echo "Example: $0 integration-testing"
    exit 1
fi

WINDOW_NAME="$1"

# Configuration
REPO_NAME="agentcontrolplane"
WORKTREES_BASE="$HOME/.humanlayer/worktrees"
TMUX_SESSION="acp-agents"

# Determine branch name from window name
BRANCH_NAME="${WINDOW_NAME}"

# Main execution
main() {
    local worktree_dir="${WORKTREES_BASE}/${REPO_NAME}_${BRANCH_NAME}"
    
    log "Cleaning up worker: $WINDOW_NAME (branch: $BRANCH_NAME)"
    
    # Kill tmux window
    if tmux has-session -t "$TMUX_SESSION" 2>/dev/null; then
        if tmux list-windows -t "$TMUX_SESSION" -F "#{window_name}" | grep -q "^${WINDOW_NAME}$"; then
            log "Killing tmux window: $TMUX_SESSION:$WINDOW_NAME"
            tmux kill-window -t "$TMUX_SESSION:$WINDOW_NAME"
        else
            info "Tmux window not found: $TMUX_SESSION:$WINDOW_NAME"
        fi
    else
        info "Tmux session not found: $TMUX_SESSION"
    fi
    
    # Remove worktree and cluster
    if [ -d "$worktree_dir" ]; then
        log "Removing worktree: $worktree_dir"
        
        # Run make teardown to clean up isolated cluster
        log "Running teardown in worktree: $worktree_dir"
        cd "$worktree_dir" 2>/dev/null && {
            if [ -f "Makefile" ]; then
                make teardown 2>/dev/null || warn "Failed to run make teardown in $worktree_dir"
            fi
            cd - > /dev/null
        }
        
        # Fix permissions before removing worktree
        log "Fixing permissions for worktree removal"
        chmod -R 755 "$worktree_dir" 2>/dev/null || warn "Failed to fix permissions for $worktree_dir"
        
        # Remove worktree
        git worktree remove --force "$worktree_dir" 2>/dev/null || {
            warn "Failed to remove worktree with git, removing directory manually"
            rm -rf "$worktree_dir"
        }
    else
        info "Worktree not found: $worktree_dir"
    fi
    
    # Delete branch
    if git show-ref --verify --quiet "refs/heads/${BRANCH_NAME}"; then
        log "Deleting branch: $BRANCH_NAME"
        git branch -D "$BRANCH_NAME" 2>/dev/null || warn "Failed to delete branch: $BRANCH_NAME"
    else
        info "Branch not found: $BRANCH_NAME"
    fi
    
    # Prune worktree list
    log "Pruning git worktree list..."
    git worktree prune
    
    log "✅ Cleanup completed successfully!"
    
    # Show remaining resources for manager visibility
    echo
    info "=== REMAINING RESOURCES ==="
    
    echo
    info "📺 Tmux sessions and windows:"
    if command -v tmux &> /dev/null && tmux list-sessions 2>/dev/null; then
        tmux list-sessions -F "Session: #{session_name}" 2>/dev/null || echo "No tmux sessions found"
        echo
        if tmux has-session -t "$TMUX_SESSION" 2>/dev/null; then
            info "Windows in $TMUX_SESSION session:"
            tmux list-windows -t "$TMUX_SESSION" -F "  #{window_index}: #{window_name}" 2>/dev/null || echo "  No windows found"
        fi
    else
        echo "No tmux sessions found"
    fi
    
    echo
    info "🌲 Git worktrees:"
    git worktree list | grep -E "(agentcontrolplane_|integration-)" || echo "No relevant worktrees found"
    
    echo
    info "🐳 Kind clusters:"
    if command -v kind &> /dev/null; then
        kind get clusters 2>/dev/null || echo "No kind clusters found"
    else
        echo "kind command not found"
    fi
    
    echo
}

# Run main
main