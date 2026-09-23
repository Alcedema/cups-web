#!/bin/bash
# Validate the explicit release decision; never create or push a tag.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
if [[ $# -ne 0 ]]; then
    echo "Automatic bumps are retired. Update release.json using docs/release-contract.md."
    echo "Run ./bump-version.sh without arguments to validate the decision."
    exit 1
fi
exec python3 scripts/release_policy.py
