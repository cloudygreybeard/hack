#!/usr/bin/env bash
# Record the hack CLI demo as animated GIF and SVG.
#
# This builds a container image with hack and demo data, then records
# the demo script running real commands inside the container.
#
# Prerequisites:
#   podman (or docker)
#   brew install asciinema agg
#   npm install -g svg-term-cli
#
# Usage:
#   ./hack/record-demo.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DOCS_DIR="${PROJECT_ROOT}/docs"
CAST_FILE="${DOCS_DIR}/hack-demo.cast"
SVG_FILE="${DOCS_DIR}/hack-demo.svg"
GIF_FILE="${DOCS_DIR}/hack-demo.gif"
IMAGE_NAME="hack-demo"

CONTAINER_RUNTIME="podman"
if ! command -v podman >/dev/null 2>&1; then
    if command -v docker >/dev/null 2>&1; then
        CONTAINER_RUNTIME="docker"
    else
        echo "Error: podman or docker is required."
        exit 1
    fi
fi

for tool in asciinema; do
    if ! command -v "${tool}" >/dev/null 2>&1; then
        echo "Error: ${tool} is not installed."
        echo "  Install with: brew install ${tool}"
        exit 1
    fi
done

mkdir -p "${DOCS_DIR}"

echo "Building demo container image..."
${CONTAINER_RUNTIME} build \
    -t "${IMAGE_NAME}" \
    -f hack/demo/Containerfile \
    "${PROJECT_ROOT}"

echo ""
echo "Recording demo..."
echo "  The demo runs real hack commands inside a container."
echo ""

if [[ -t 0 ]]; then
    echo "Press Enter to start recording..."
    read -r
fi

asciinema rec "${CAST_FILE}" \
    --command "${CONTAINER_RUNTIME} run --rm -it -e TYPE_SPEED=${TYPE_SPEED:-0.020} ${IMAGE_NAME}" \
    --idle-time-limit 3 \
    --overwrite \
    --output-format asciicast-v2 \
    --cols 90 \
    --rows 30

echo ""

if command -v agg >/dev/null 2>&1; then
    echo "Generating GIF with agg..."
    agg "${CAST_FILE}" "${GIF_FILE}" \
        --cols 90 \
        --rows 30 \
        --font-size 16
fi

if command -v svg-term >/dev/null 2>&1; then
    echo "Generating SVG with svg-term..."
    svg-term \
        --in "${CAST_FILE}" \
        --out "${SVG_FILE}" \
        --window \
        --width 90 \
        --height 30 \
        --padding 10
fi

echo ""
echo "Recording complete."
echo "  Cast: ${CAST_FILE} ($(du -h "${CAST_FILE}" | cut -f1))"
[[ -f "${GIF_FILE}" ]] && echo "  GIF:  ${GIF_FILE} ($(du -h "${GIF_FILE}" | cut -f1))"
[[ -f "${SVG_FILE}" ]] && echo "  SVG:  ${SVG_FILE} ($(du -h "${SVG_FILE}" | cut -f1))"
echo ""
echo "To include in README.md:"
[[ -f "${GIF_FILE}" ]] && echo "  ![hack CLI Demo](docs/hack-demo.gif)"
[[ -f "${SVG_FILE}" ]] && echo "  ![hack CLI Demo](docs/hack-demo.svg)"
