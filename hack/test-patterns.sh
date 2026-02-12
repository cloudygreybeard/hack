#!/usr/bin/env bash
# Pattern integration tests for hack
#
# Tests pattern creation, variable substitution, and security validation.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"
WORKSPACE_DIR="$(dirname "$REPO_DIR")"
HACK_BIN="$REPO_DIR/bin/hack"
TEST_DIR="/tmp/hack-pattern-tests-$$"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

passed=0
failed=0

log_pass() {
    echo -e "${GREEN}PASS${NC}: $1"
    passed=$((passed + 1))
}

log_fail() {
    echo -e "${RED}FAIL${NC}: $1"
    failed=$((failed + 1))
}

log_info() {
    echo -e "${YELLOW}INFO${NC}: $1"
}

# Setup test environment
setup() {
    log_info "Setting up test environment at $TEST_DIR"
    mkdir -p "$TEST_DIR"
    
    # Install patterns from workspace
    for p in "$WORKSPACE_DIR"/patterns/*/; do
        name=$(basename "$p")
        log_info "Installing pattern: $name"
        "$HACK_BIN" pattern install "$p" >/dev/null
    done
    
    # Override HACK_ROOT_DIR for tests
    export HACK_ROOT_DIR="$TEST_DIR"
}

# Cleanup test environment
cleanup() {
    log_info "Cleaning up $TEST_DIR"
    rm -rf "$TEST_DIR"
}

trap cleanup EXIT

# Test: Basic pattern creation (aro-refs)
test_aro_refs_basic() {
    local name="test-aro-refs"
    log_info "Testing: aro-refs basic creation"
    
    if "$HACK_BIN" create "$name" -p aro-refs --no-git --no-edit -q >/dev/null 2>&1; then
        local ws_dir
        ws_dir=$(find "$TEST_DIR" -maxdepth 1 -type d -name "*.$name" | head -1)
        
        if [[ -f "$ws_dir/README.md" ]] && [[ -f "$ws_dir/CLAUDE.md" ]]; then
            log_pass "aro-refs basic creation"
        else
            log_fail "aro-refs basic creation - missing files"
        fi
    else
        log_fail "aro-refs basic creation - command failed"
    fi
}

# Test: aro-local-dev pattern with setup script
test_aro_local_dev() {
    local name="test-aro-local-dev"
    log_info "Testing: aro-local-dev creation"
    
    if "$HACK_BIN" create "$name" -p aro-local-dev --no-git --no-edit -q >/dev/null 2>&1; then
        local ws_dir
        ws_dir=$(find "$TEST_DIR" -maxdepth 1 -type d -name "*.$name" | head -1)
        
        if [[ -f "$ws_dir/hack/setup.sh" ]] && [[ -x "$ws_dir/hack/setup.sh" || -f "$ws_dir/hack/setup.sh" ]]; then
            # Check setup.sh contains expected branch
            if grep -q "master" "$ws_dir/hack/setup.sh"; then
                log_pass "aro-local-dev creation with setup script"
            else
                log_fail "aro-local-dev creation - setup.sh missing branch"
            fi
        else
            log_fail "aro-local-dev creation - missing setup.sh"
        fi
    else
        log_fail "aro-local-dev creation - command failed"
    fi
}

# Test: go-cli pattern with app subdirectory
test_go_cli_basic() {
    local name="test-go-cli"
    log_info "Testing: go-cli basic creation"
    
    if "$HACK_BIN" create "$name" -p go-cli --no-git --no-edit -q >/dev/null 2>&1; then
        local ws_dir
        ws_dir=$(find "$TEST_DIR" -maxdepth 1 -type d -name "*.$name" | head -1)
        
        # Check workspace-level files
        if [[ -f "$ws_dir/README.md" ]] && [[ -f "$ws_dir/CLAUDE.md" ]]; then
            # Check app subdirectory
            if [[ -d "$ws_dir/$name" ]] && [[ -f "$ws_dir/$name/main.go" ]] && [[ -f "$ws_dir/$name/Makefile" ]]; then
                log_pass "go-cli basic creation"
            else
                log_fail "go-cli basic creation - missing app files"
            fi
        else
            log_fail "go-cli basic creation - missing workspace files"
        fi
    else
        log_fail "go-cli basic creation - command failed"
    fi
}

# Test: go-cli with custom app name
test_go_cli_custom_app() {
    local name="test-go-cli-custom"
    local app_name="myapp"
    log_info "Testing: go-cli with custom app name"
    
    if "$HACK_BIN" create "$name" -p go-cli --app-name "$app_name" --no-git --no-edit -q >/dev/null 2>&1; then
        local ws_dir
        ws_dir=$(find "$TEST_DIR" -maxdepth 1 -type d -name "*.$name" | head -1)
        
        if [[ -d "$ws_dir/$app_name" ]] && [[ -f "$ws_dir/$app_name/main.go" ]]; then
            # Check go.mod has correct module name
            if grep -q "$app_name" "$ws_dir/$app_name/go.mod"; then
                log_pass "go-cli with custom app name"
            else
                log_fail "go-cli custom app - go.mod has wrong module"
            fi
        else
            log_fail "go-cli custom app - missing app directory"
        fi
    else
        log_fail "go-cli custom app - command failed"
    fi
}

# Test: Adding second app to existing workspace
test_add_app_to_workspace() {
    local name="test-multi-app"
    log_info "Testing: adding second app to workspace"
    
    # Create first app
    if ! "$HACK_BIN" create "$name" -p go-cli --app-name first-app --no-git --no-edit -q >/dev/null 2>&1; then
        log_fail "multi-app - first app creation failed"
        return
    fi
    
    # Add second app
    if "$HACK_BIN" create "$name" -p go-cli --app-name second-app --no-git --no-edit -q >/dev/null 2>&1; then
        local ws_dir
        ws_dir=$(find "$TEST_DIR" -maxdepth 1 -type d -name "*.$name" | head -1)
        
        if [[ -d "$ws_dir/first-app" ]] && [[ -d "$ws_dir/second-app" ]]; then
            log_pass "adding second app to workspace"
        else
            log_fail "multi-app - missing app directories"
        fi
    else
        log_fail "multi-app - second app creation failed"
    fi
}

# Test: Security - path traversal rejection
test_security_path_traversal() {
    log_info "Testing: security - path traversal rejection"
    
    if "$HACK_BIN" create "../escape" -p aro-refs --no-git --no-edit -q >/dev/null 2>&1; then
        log_fail "security - path traversal was NOT rejected"
    else
        log_pass "security - path traversal rejected"
    fi
}

# Test: Security - path separator rejection
test_security_path_separator() {
    log_info "Testing: security - path separator rejection"
    
    if "$HACK_BIN" create "bad/name" -p aro-refs --no-git --no-edit -q >/dev/null 2>&1; then
        log_fail "security - path separator was NOT rejected"
    else
        log_pass "security - path separator rejected"
    fi
}

# Test: Verbose output shows file creation
test_verbose_output() {
    local name="test-verbose"
    log_info "Testing: verbose output"
    
    local output
    output=$("$HACK_BIN" create "$name" -p aro-refs --no-git --no-edit -v 2>&1) || true
    
    if echo "$output" | grep -q "create:"; then
        log_pass "verbose output shows file creation"
    else
        log_fail "verbose output missing file creation info"
    fi
}

# Main
main() {
    echo "========================================"
    echo "Hack Pattern Integration Tests"
    echo "========================================"
    echo ""
    
    setup
    echo ""
    
    # Run tests
    test_aro_refs_basic
    test_aro_local_dev
    test_go_cli_basic
    test_go_cli_custom_app
    test_add_app_to_workspace
    test_security_path_traversal
    test_security_path_separator
    test_verbose_output
    
    echo ""
    echo "========================================"
    echo -e "Results: ${GREEN}$passed passed${NC}, ${RED}$failed failed${NC}"
    echo "========================================"
    
    if [[ $failed -gt 0 ]]; then
        exit 1
    fi
}

main "$@"
