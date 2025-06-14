#!/bin/bash
# cleanup_coding_workers.sh - Cleans up worktree environments and tmux sessions
# Usage: ./cleanup_coding_workers.sh [suffix] [--tmux-only|--worktrees-only]

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
SUFFIX="${1:-}"
CLEANUP_MODE="${2:-all}"

# Configuration
REPO_NAME="agentcontrolplane"
WORKTREES_BASE="$HOME/.humanlayer/worktrees"

# Define branch names based on suffix
if [ -n "$SUFFIX" ]; then
    TMUX_SESSION="acp-coding-$SUFFIX"
    declare -a BRANCH_NAMES=(
        "acp-srs-$SUFFIX"
        "acp-projectid-$SUFFIX"
        "acp-taskspec-$SUFFIX"
        "acp-channelapikey-$SUFFIX"
        "acp-v1beta3-$SUFFIX"
        "acp-parallel-$SUFFIX"
        "acp-merge-$SUFFIX"
    )
else
    TMUX_SESSION=""
    declare -a BRANCH_NAMES=()
fi

# Function to kill tmux session
cleanup_tmux() {
    if [ -z "$SUFFIX" ]; then
        warn "No suffix provided, cleaning up all acp-coding-* sessions"
        local sessions=$(tmux list-sessions 2>/dev/null | grep "^acp-coding-" | cut -d: -f1 || true)
        if [ -z "$sessions" ]; then
            info "No acp-coding-* tmux sessions found"
        else
            for session in $sessions; do
                log "Killing tmux session: $session"
                tmux kill-session -t "$session" 2>/dev/null || warn "Session $session not found"
            done
        fi
    else
        if tmux has-session -t "$TMUX_SESSION" 2>/dev/null; then
            log "Killing tmux session: $TMUX_SESSION"
            tmux kill-session -t "$TMUX_SESSION"
        else
            info "Tmux session not found: $TMUX_SESSION"
        fi
    fi
}

# Function to remove worktree
remove_worktree() {
    local branch_name=$1
    local worktree_dir="${WORKTREES_BASE}/${REPO_NAME}_${branch_name}"
    
    if [ -d "$worktree_dir" ]; then
        log "Removing worktree: $worktree_dir"
        git worktree remove --force "$worktree_dir" 2>/dev/null || {
            warn "Failed to remove worktree with git, removing directory manually"
            rm -rf "$worktree_dir"
        }
    else
        info "Worktree not found: $worktree_dir"
    fi
}

# Function to delete branch
delete_branch() {
    local branch_name=$1
    
    if git show-ref --verify --quiet "refs/heads/${branch_name}"; then
        log "Deleting branch: $branch_name"
        git branch -D "$branch_name" 2>/dev/null || warn "Failed to delete branch: $branch_name"
    else
        info "Branch not found: $branch_name"
    fi
}

# Function to cleanup all worktrees
cleanup_worktrees() {
    if [ -z "$SUFFIX" ]; then
        warn "No suffix provided, cleaning up all acp-* worktrees"
        if [ -d "$WORKTREES_BASE" ]; then
            local worktrees=$(ls "$WORKTREES_BASE" | grep "^${REPO_NAME}_acp-" || true)
            if [ -z "$worktrees" ]; then
                info "No acp-* worktrees found"
            else
                for worktree in $worktrees; do
                    local branch_name="${worktree#${REPO_NAME}_}"
                    remove_worktree "$branch_name"
                    delete_branch "$branch_name"
                done
            fi
        fi
    else
        for branch_name in "${BRANCH_NAMES[@]}"; do
            remove_worktree "$branch_name"
            delete_branch "$branch_name"
        done
    fi
    
    # Prune worktree list
    log "Pruning git worktree list..."
    git worktree prune
}

# Function to show usage
usage() {
    echo "Usage: $0 [suffix] [--tmux-only|--worktrees-only]"
    echo
    echo "Options:"
    echo "  suffix          - The suffix used when launching workers (optional)"
    echo "  --tmux-only     - Only clean up tmux sessions"
    echo "  --worktrees-only - Only clean up worktrees and branches"
    echo
    echo "If no suffix is provided, will clean up all acp-* sessions and worktrees"
    echo
    echo "Examples:"
    echo "  $0                    # Clean up all acp-* sessions and worktrees"
    echo "  $0 1234              # Clean up specific suffix"
    echo "  $0 1234 --tmux-only  # Only clean up tmux for suffix 1234"
}

# Main execution
main() {
    log "Starting cleanup_coding_workers.sh"
    
    if [ "$CLEANUP_MODE" == "--help" ] || [ "$CLEANUP_MODE" == "-h" ]; then
        usage
        exit 0
    fi
    
    # Status report before cleanup
    info "=== Current Status ==="
    echo "Tmux sessions:"
    tmux list-sessions 2>/dev/null | grep "acp-coding-" || echo "  None found"
    echo
    echo "Git worktrees:"
    git worktree list | grep -E "acp-|merge-" || echo "  None found"
    echo
    
    # Perform cleanup based on mode
    case "$CLEANUP_MODE" in
        --tmux-only)
            log "Cleaning up tmux sessions only..."
            cleanup_tmux
            ;;
        --worktrees-only)
            log "Cleaning up worktrees only..."
            cleanup_worktrees
            ;;
        all|"")
            log "Cleaning up everything..."
            cleanup_tmux
            cleanup_worktrees
            ;;
        *)
            error "Unknown cleanup mode: $CLEANUP_MODE"
            usage
            exit 1
            ;;
    esac
    
    # Status report after cleanup
    info "=== Status After Cleanup ==="
    echo "Tmux sessions:"
    tmux list-sessions 2>/dev/null | grep "acp-coding-" || echo "  None found"
    echo
    echo "Git worktrees:"
    git worktree list | grep -E "acp-|merge-" || echo "  None found"
    echo
    
    log "✅ Cleanup completed successfully!"
}

# Run main
main