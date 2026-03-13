#!/usr/bin/env bash
# Seed demo workspaces with labels for hack list demonstrations.
# Run during container build to pre-populate ~/hack/.

set -e

HACK_ROOT="${HOME}/hack"
mkdir -p "${HACK_ROOT}"

create_workspace() {
    local dir_name="$1"
    shift
    local ws_dir="${HACK_ROOT}/${dir_name}"
    local ws_name
    ws_name="${dir_name#*.}"

    mkdir -p "${ws_dir}"
    git -C "${ws_dir}" init --quiet

    cat > "${ws_dir}/README.md" <<EOF
# ${ws_name}

A demo workspace.
EOF

    local labels_yaml=""
    local has_labels=false
    for label in "$@"; do
        local key="${label%%=*}"
        local value="${label#*=}"
        labels_yaml="${labels_yaml}    ${key}: ${value}\n"
        has_labels=true
    done

    {
        echo "apiVersion: hack/v1"
        echo "kind: Workspace"
        echo "metadata:"
        echo "  name: ${ws_name}"
        if [[ "${has_labels}" == "true" ]]; then
            echo "  labels:"
            printf "%b" "${labels_yaml}"
        fi
    } > "${ws_dir}/.hack.yaml"
}

create_workspace "2026-01-20.api-gateway" "domain=backend" "lang=go"
create_workspace "2026-01-22.auth-service" "domain=backend" "lang=go"
create_workspace "2026-01-24.dashboard-ui" "domain=frontend" "lang=ts"
