#!/usr/bin/env bash
# fetch-vlt-skill.sh -- Download and install the vlt skill from GitHub.
#
# Fetches the vlt skill from a pinned release of the RamXX/vlt repository
# and installs it to the user's global Claude Code skills directory.
# Verifies the download with SHA256 checksum.
#
# Usage:
#   fetch-vlt-skill.sh              Install if missing, skip if present.
#   fetch-vlt-skill.sh --force      Always re-download and overwrite.

set -euo pipefail

REPO="RamXX/vlt"
VERSION="v0.9.0"
EXPECTED_SHA256="cf91fe6ea74b97d3da982c3ee402b8df2abeedc7725a4324b74f2da04d4fa888"
SKILL_DIR="$HOME/.claude/skills/vlt-skill"
FORCE=false

if [[ "${1:-}" == "--force" ]]; then
    FORCE=true
fi

# ---------------------------------------------------------------------------
# Skip if already installed (unless --force)
# ---------------------------------------------------------------------------
if [ -f "$SKILL_DIR/SKILL.md" ] && [ "$FORCE" = false ]; then
    echo "vlt skill already installed at $SKILL_DIR (use --force to update)"
    exit 0
fi

# ---------------------------------------------------------------------------
# Download, verify, and extract
# ---------------------------------------------------------------------------
TARBALL_URL="https://github.com/${REPO}/archive/refs/tags/${VERSION}.tar.gz"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Fetching vlt skill from github.com/${REPO} @ ${VERSION}..."
if ! curl -sSfL "$TARBALL_URL" -o "$TMP_DIR/vlt.tar.gz"; then
    echo "ERROR: Failed to download from $TARBALL_URL" >&2
    echo "       Check your internet connection or install the vlt skill manually:" >&2
    echo "       git clone https://github.com/${REPO}.git && cd vlt && make install-skill" >&2
    exit 1
fi

# Verify checksum
ACTUAL_SHA256=$(shasum -a 256 "$TMP_DIR/vlt.tar.gz" | cut -d' ' -f1)
if [ "$ACTUAL_SHA256" != "$EXPECTED_SHA256" ]; then
    echo "ERROR: SHA256 checksum mismatch!" >&2
    echo "  Expected: $EXPECTED_SHA256" >&2
    echo "  Got:      $ACTUAL_SHA256" >&2
    echo "  The downloaded file may have been tampered with." >&2
    exit 1
fi

# Extract just the skill directory
STRIP_PREFIX="vlt-${VERSION#v}"
mkdir -p "$TMP_DIR/extracted"
tar xzf "$TMP_DIR/vlt.tar.gz" --strip-components=3 -C "$TMP_DIR/extracted" "${STRIP_PREFIX}/docs/vlt-skill"

# ---------------------------------------------------------------------------
# Install to global skills directory
# ---------------------------------------------------------------------------
mkdir -p "$(dirname "$SKILL_DIR")"
rm -rf "$SKILL_DIR"
mv "$TMP_DIR/extracted" "$SKILL_DIR"

echo "Installed vlt skill to $SKILL_DIR (${VERSION}, verified)"
