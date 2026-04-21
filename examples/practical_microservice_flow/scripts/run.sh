#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
VENV_DIR="$ROOT_DIR/examples/practical_microservice_flow/.venv"

pushd "$ROOT_DIR" >/dev/null

python3 -m venv "$VENV_DIR"
"$VENV_DIR/bin/python" -m pip install --upgrade pip >/dev/null
"$VENV_DIR/bin/python" -m pip install -r "$ROOT_DIR/examples/practical_microservice_flow/requirements.txt" >/dev/null

PYTHON_BIN="$VENV_DIR/bin/python" go run ./examples/practical_microservice_flow/service2_consumer

popd >/dev/null
