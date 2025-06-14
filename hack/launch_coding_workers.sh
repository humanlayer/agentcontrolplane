#!/bin/bash
# launch_coding_workers.sh - Sets up parallel work environments for executing code
# Usage: ./launch_coding_workers.sh [suffix]

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

# Get suffix argument
SUFFIX="${1:-$(date +%s)}"
log "Using suffix: $SUFFIX"

# Configuration
REPO_NAME="agentcontrolplane"
WORKTREES_BASE="$HOME/.humanlayer/worktrees"
TMUX_SESSION="acp-coding-$SUFFIX"

# Define plan files and their configurations
declare -a PLAN_FILES=(
    "plan-srs-implementation.md"
    "plan-contactchannel-projectid.md"
    "plan-contactchannel-taskspec.md"
    "plan-channel-apikey-id.md"
    "plan-v1beta3-events.md"
    "plan-parallel-llm-calls-fix.md"
    "plan-kustomization-template.md"
)

declare -a BRANCH_NAMES=(
    "acp-srs-$SUFFIX"
    "acp-projectid-$SUFFIX"
    "acp-taskspec-$SUFFIX"
    "acp-channelapikey-$SUFFIX"
    "acp-v1beta3-$SUFFIX"
    "acp-parallel-$SUFFIX"
    "acp-kustomize-$SUFFIX"
)

# Merge agent configuration
MERGE_PLAN="plan-merge-agent.md"
MERGE_BRANCH="acp-merge-$SUFFIX"

# Function to create worktree
create_worktree() {
    local branch_name=$1
    local plan_file=$2
    local worktree_dir="${WORKTREES_BASE}/${REPO_NAME}_${branch_name}"
    
    log "Creating worktree for $branch_name..."
    
    # Use create_worktree.sh if available
    if [ -f "hack/create_worktree.sh" ]; then
        ./hack/create_worktree.sh "$branch_name"
    else
        # Fallback to manual creation
        if [ ! -d "$WORKTREES_BASE" ]; then
            mkdir -p "$WORKTREES_BASE"
        fi
        
        if [ -d "$worktree_dir" ]; then
            warn "Worktree already exists: $worktree_dir"
            return 0
        fi
        
        git worktree add -b "$branch_name" "$worktree_dir" HEAD
        
        # Copy .claude directory
        if [ -d ".claude" ]; then
            cp -r .claude "$worktree_dir/"
        fi
    fi
    
    # Copy plan file
    cp "$plan_file" "$worktree_dir/"
    
    # Create prompt.md file
    cat > "$worktree_dir/prompt.md" << EOF
Adopt the persona from hack/agent-developer.md

Your task is to implement the features described in $plan_file

Key requirements:
- Read and understand the plan in $plan_file
- Follow the Dan Abramov methodology
- Commit your changes every 5-10 minutes
- Run tests frequently
- Delete more code than you add
- Keep a 20+ item TODO list

Start by reading the plan file and understanding the task ahead.
EOF
    
    log "Worktree created: $worktree_dir"
}

# Function to launch tmux window for agent
launch_agent_window() {
    local window_num=$1
    local branch_name=$2
    local plan_file=$3
    local window_name=$(basename "$plan_file" .md)
    local worktree_dir="${WORKTREES_BASE}/${REPO_NAME}_${branch_name}"
    
    log "Launching window $window_num: $window_name"
    
    # Create window
    if [ "$window_num" -eq 1 ]; then
        tmux new-session -d -s "$TMUX_SESSION" -n "$window_name" -c "$worktree_dir"
    else
        tmux new-window -t "$TMUX_SESSION:$window_num" -n "$window_name" -c "$worktree_dir"
    fi
    
    # Split window horizontally
    tmux split-window -t "$TMUX_SESSION:$window_num" -v -c "$worktree_dir"
    
    # Top pane: Troubleshooting terminal (pane 1)
    tmux send-keys -t "$TMUX_SESSION:$window_num.1" "echo 'Troubleshooting terminal for $window_name'" C-m
    tmux send-keys -t "$TMUX_SESSION:$window_num.1" "echo 'Branch: $branch_name'" C-m
    tmux send-keys -t "$TMUX_SESSION:$window_num.1" "git status" C-m
    
    # Bottom pane: Claude Code (pane 2, with focus)
    tmux select-pane -t "$TMUX_SESSION:$window_num.2"
    tmux send-keys -t "$TMUX_SESSION:$window_num.2" "claude \"\$(cat prompt.md)\"" C-m
    # Send newline to accept trust directory prompt
    sleep 1
    tmux send-keys -t "$TMUX_SESSION:$window_num.2" C-m
}

# Main execution
main() {
    log "Starting launch_coding_workers.sh with suffix: $SUFFIX"
    
    # Check prerequisites
    if ! command -v tmux &> /dev/null; then
        error "tmux is not installed"
        exit 1
    fi
    
    if ! command -v claude &> /dev/null; then
        error "claude CLI is not installed"
        exit 1
    fi
    
    # Kill existing session if it exists
    if tmux has-session -t "$TMUX_SESSION" 2>/dev/null; then
        warn "Killing existing tmux session: $TMUX_SESSION"
        tmux kill-session -t "$TMUX_SESSION"
    fi
    
    # Create worktrees for all agents
    log "Creating worktrees..."
    for i in "${!PLAN_FILES[@]}"; do
        create_worktree "${BRANCH_NAMES[$i]}" "${PLAN_FILES[$i]}"
    done
    
    # Create merge agent worktree
    log "Creating merge agent worktree..."
    create_worktree "$MERGE_BRANCH" "$MERGE_PLAN"
    
    # Create merge agent prompt
    local merge_worktree="${WORKTREES_BASE}/${REPO_NAME}_${MERGE_BRANCH}"
    cat > "$merge_worktree/prompt.md" << EOF
Adopt the persona from hack/agent-merger.md

Your task is to merge the work from the following branches into the current branch:
${BRANCH_NAMES[@]}

Key requirements:
- Read the plan in $MERGE_PLAN
- Monitor agent branches for commits every 2 minutes
- Merge changes in dependency order
- Resolve conflicts appropriately
- Maintain clean build state
- Commit merged changes

Start by reading the merge plan and checking the status of all agent branches.
EOF
    
    # Launch agent windows
    log "Launching tmux session: $TMUX_SESSION"
    for i in "${!PLAN_FILES[@]}"; do
        launch_agent_window $((i+1)) "${BRANCH_NAMES[$i]}" "${PLAN_FILES[$i]}"
    done
    
    # Launch merge agent in the last window
    local merge_window=$((${#PLAN_FILES[@]} + 1))
    launch_agent_window "$merge_window" "$MERGE_BRANCH" "$MERGE_PLAN"
    
    # Summary
    log "✅ All coding workers launched successfully!"
    echo
    echo "Session: $TMUX_SESSION"
    echo "Agents:"
    for i in "${!PLAN_FILES[@]}"; do
        echo "  - Window $((i+1)): ${BRANCH_NAMES[$i]} (${PLAN_FILES[$i]})"
    done
    echo "  - Window $merge_window: $MERGE_BRANCH (merge agent)"
    echo
    echo "To attach to the session:"
    echo "  tmux attach -t $TMUX_SESSION"
    echo
    echo "To switch between windows:"
    echo "  Ctrl-b [window-number]"
    echo
    echo "To clean up later:"
    echo "  ./cleanup_coding_workers.sh $SUFFIX"
}

# Run main
main