#!/usr/bin/env bash
# hack CLI demo script
# Runs real hack commands with a typing animation for terminal recording.
# Designed to run inside the demo container built from hack/demo/Containerfile.

set -e

YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

TYPE_SPEED="${TYPE_SPEED:-0.020}"

type_command() {
    local cmd="$1"
    local speed="${2:-$TYPE_SPEED}"
    printf "${BLUE}\$ ${NC}"
    for (( i=0; i<${#cmd}; i++ )); do
        printf "%s" "${cmd:$i:1}"
        sleep "$speed"
    done
    printf "\n"
}

run_cmd() {
    local cmd="$1"
    type_command "${cmd}"
    eval "${cmd}"
}

# Type one command for display, execute another
run_as() {
    local display="$1"
    local actual="$2"
    type_command "${display}"
    eval "${actual}"
}

demo_pause() {
    sleep "${1:-1.5}"
}

comment() {
    printf "\n${YELLOW}# %s${NC}\n" "$1"
    demo_pause 0.8
}

clear

comment "hack - development workspace manager"
demo_pause 2

# --- Version ---
comment "Check the version"
run_cmd "hack version"
demo_pause 2

# --- Pattern list ---
comment "List available patterns"
run_cmd "hack pattern list"
demo_pause 3

# --- Create workspace with pattern ---
comment "Create a workspace with a pattern"
run_cmd "hack create my-cli -p go-cli --no-edit"
demo_pause 2

# Discover the created workspace directory
WS_DIR=$(find ~/hack -maxdepth 1 -type d -name '*.my-cli' | head -1)
WS_NAME=$(basename "${WS_DIR}")

# --- Show created files ---
comment "Show what was created"
run_as "ls ~/hack/${WS_NAME}/" "ls ${WS_DIR}/"
demo_pause 2

# --- Show metadata ---
comment "Workspace metadata is stored in .hack.yaml"
run_as "yq -P ~/hack/${WS_NAME}/.hack.yaml" "yq -P ${WS_DIR}/.hack.yaml"
demo_pause 3

# --- Add labels ---
comment "Add labels to a workspace"
run_cmd "hack label my-cli domain=tools priority=high"
demo_pause 2

# --- List with labels ---
comment "List workspaces with labels"
run_cmd "hack list --show-labels"
demo_pause 3

# --- Filtered list ---
comment "Filter by label selector"
run_cmd "hack list -l domain=backend"
demo_pause 2

# --- Pattern show ---
comment "Show pattern details"
run_cmd "hack pattern show go-cli"
demo_pause 3

# --- Dry-run ---
comment "Preview a create with --dry-run"
run_cmd "hack create new-service -p go-service --label team=platform --dry-run --no-edit"
demo_pause 3

# --- Archive ---
comment "Archive a workspace when done"
run_as "hack archive dashboard-ui" "echo y | hack archive dashboard-ui"
demo_pause 2

# --- List after archive ---
comment "Verify it was archived"
run_cmd "hack list"
demo_pause 3

comment "Done. See github.com/cloudygreybeard/hack for more."
demo_pause 3

if [[ -t 0 ]]; then
    read -n 1 -s -r
fi
