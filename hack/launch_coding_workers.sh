#!/bin/bash
# launch_coding_workers.sh - Launches a single coding agent with dedicated worktree and cluster
# Usage: ./launch_coding_workers.sh <branch_name> <plan_file>

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

# Parse arguments
if [ $# -ne 2 ]; then
    echo "Usage: $0 <branch_name> <plan_file>"
    echo "Example: $0 integration-testing plan-integration-testing.md"
    exit 1
fi

BRANCH_NAME="$1"
PLAN_FILE="$2"

# Configuration
REPO_NAME="agentcontrolplane"
WORKTREES_BASE="$HOME/.humanlayer/worktrees"
TMUX_SESSION="acp-agents"

# Function to create worktree
create_worktree() {
    local worktree_dir="${WORKTREES_BASE}/${REPO_NAME}_${BRANCH_NAME}"
    
    log "Creating worktree for $BRANCH_NAME..."
    
    # Create worktrees directory
    if [ ! -d "$WORKTREES_BASE" ]; then
        mkdir -p "$WORKTREES_BASE"
    fi
    
    # Remove existing worktree if it exists
    if [ -d "$worktree_dir" ]; then
        warn "Removing existing worktree: $worktree_dir"
        git worktree remove --force "$worktree_dir" 2>/dev/null || rm -rf "$worktree_dir"
    fi
    
    # Create new worktree
    git worktree add -b "$BRANCH_NAME" "$worktree_dir" HEAD
    
    # Copy .claude directory
    if [ -d ".claude" ]; then
        cp -r .claude "$worktree_dir/"
    fi
    
    # Copy plan file
    cp "$PLAN_FILE" "$worktree_dir/"
    
    # Run make setup to create isolated cluster
    log "Setting up isolated cluster in worktree..."
    cd "$worktree_dir"
    if ! make setup; then
        error "Setup failed. Cleaning up worktree..."
        cd - > /dev/null
        git worktree remove --force "$worktree_dir" 2>/dev/null || rm -rf "$worktree_dir"
        git branch -D "$BRANCH_NAME" 2>/dev/null || true
        exit 1
    fi
    cd - > /dev/null
    
    # Create prompt.md file based on plan type
    if [[ "$PLAN_FILE" == "hack/agent-integration-tester.md" ]]; then
        # Copy the integration tester persona directly as the prompt
        cp hack/agent-integration-tester.md "$worktree_dir/prompt.md"
    else
        # Copy the plan file as the prompt for regular agents
        cp "$PLAN_FILE" "$worktree_dir/prompt.md"
    fi
    
    log "Worktree created: $worktree_dir"
}

# Main execution
main() {
    local worktree_dir="${WORKTREES_BASE}/${REPO_NAME}_${BRANCH_NAME}"
    local window_name="$BRANCH_NAME"
    
    log "Starting single worker: $BRANCH_NAME with plan: $PLAN_FILE"
    
    # Check prerequisites
    if ! command -v tmux &> /dev/null; then
        error "tmux is not installed"
        exit 1
    fi
    
    if ! command -v claude &> /dev/null; then
        error "claude CLI is not installed"
        exit 1
    fi
    
    if [ ! -f "$PLAN_FILE" ]; then
        error "Plan file not found: $PLAN_FILE"
        exit 1
    fi
    
    # Create worktree
    create_worktree
    
    # Create session if it doesn't exist, otherwise add new window
    local new_window=""
    if tmux has-session -t "$TMUX_SESSION" 2>/dev/null; then
        # Find the highest window number and add 1
        local max_window=$(tmux list-windows -t "$TMUX_SESSION" -F "#{window_index}" | sort -n | tail -1)
        new_window=$((max_window + 1))
        log "Adding new window to existing session: $TMUX_SESSION (window $new_window)"
        tmux new-window -t "$TMUX_SESSION:$new_window" -n "$window_name" -c "$worktree_dir"
    else
        log "Creating new tmux session: $TMUX_SESSION"
        tmux new-session -d -s "$TMUX_SESSION" -n "$window_name" -c "$worktree_dir"
    fi
    
    # Set KUBECONFIG environment variable for the tmux window
    local target_window=""
    if [ -n "${new_window:-}" ]; then
        target_window="$TMUX_SESSION:$new_window"
    else
        target_window="$TMUX_SESSION:$window_name"
    fi
    
    log "Setting KUBECONFIG for isolated cluster"
    tmux send-keys -t "$target_window" "export KUBECONFIG=\"$worktree_dir/.kube/config\"" C-m
    
    # Launch Claude Code in the current window
    log "Starting Claude Code in worktree: $worktree_dir"
    tmux send-keys -t "$target_window" 'claude "$(cat prompt.md)"' C-m
    sleep 2
    # Send multiple C-m to handle Claude trust prompt confirmation
    tmux send-keys -t "$target_window" C-m
    sleep 1
    tmux send-keys -t "$target_window" C-m
    sleep 1
    # Send Shift+Tab to enable auto-accept edits mode
    tmux send-keys -t "$target_window" S-Tab
    
    # Summary
    log "✅ Worker launched successfully!"
    echo
    echo "Session: $TMUX_SESSION"
    echo "Branch: $BRANCH_NAME"
    echo "Plan: $PLAN_FILE"
    echo "Worktree: $worktree_dir"
    echo
    echo "To attach to the session:"
    echo "  tmux attach -t $TMUX_SESSION"
    echo
    echo "To switch to this window:"
    echo "  tmux select-window -t $TMUX_SESSION:$window_name"
    echo
    echo "To clean up later:"
    echo "  ./cleanup_coding_workers.sh $window_name"
}

# Run main
main