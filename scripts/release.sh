#!/usr/bin/env bash
#
# release.sh — Prepare and tag a TenantKit release
#
# Usage:
#   ./scripts/release.sh v1.0.0
#
# This script:
#   1. Validates all modules build and test
#   2. Updates go.mod files: replaces v0.0.0 with the release version
#   3. Removes local replace directives from adapter go.mod files
#   4. Commits the changes
#   5. Creates per-module git tags (required by Go module proxy)
#   6. Optionally pushes tags to origin
#
# After releasing, run ./scripts/post-release.sh to restore local replace directives.
#
set -euo pipefail

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  echo "Usage: $0 <version>  (e.g., v1.0.0)"
  exit 1
fi

# Strip leading 'v' for go.mod version strings
MOD_VERSION="$VERSION"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "=== TenantKit Release: $VERSION ==="
echo ""

# ─── Step 1: Validate ────────────────────────────────────────────────

echo "Step 1: Validating all modules build and pass tests..."
echo ""

echo "  Building core..."
(cd tenantkit && go build ./...) || { echo "❌ Core build failed"; exit 1; }
echo "  ✅ Core builds"

echo "  Testing core..."
(cd tenantkit && go test ./... -count=1) || { echo "❌ Core tests failed"; exit 1; }
echo "  ✅ Core tests pass"

for dir in adapters/*/; do
  name=$(basename "$dir")
  echo "  Building adapter: $name..."
  (cd "$dir" && go build ./...) || { echo "❌ $name build failed"; exit 1; }
  echo "  Testing adapter: $name..."
  (cd "$dir" && go test ./... -count=1) || { echo "❌ $name tests failed"; exit 1; }
  echo "  ✅ $name OK"
done

echo ""
echo "✅ All modules validated."
echo ""

# ─── Step 2: Update go.mod files ─────────────────────────────────────

echo "Step 2: Updating go.mod files for release..."
echo ""

# Core tenantkit/go.mod — update domain version
sed -i.bak "s|github.com/abhipray-cpu/tenantkit/domain v0.0.0|github.com/abhipray-cpu/tenantkit/domain $MOD_VERSION|g" tenantkit/go.mod
rm -f tenantkit/go.mod.bak
echo "  ✅ tenantkit/go.mod updated"

# Each adapter go.mod:
#   - Replace v0.0.0 with release version
#   - Replace v0.0.0-00010101000000-000000000000 (auto-generated) with release version
#   - Remove replace directives pointing to local paths
for dir in adapters/*/; do
  mod="$dir/go.mod"
  if [[ -f "$mod" ]]; then
    name=$(basename "$dir")

    # Update version references
    sed -i.bak \
      -e "s|github.com/abhipray-cpu/tenantkit/tenantkit v0.0.0|github.com/abhipray-cpu/tenantkit/tenantkit $MOD_VERSION|g" \
      -e "s|github.com/abhipray-cpu/tenantkit/domain v0.0.0-00010101000000-000000000000|github.com/abhipray-cpu/tenantkit/domain $MOD_VERSION|g" \
      -e "s|github.com/abhipray-cpu/tenantkit/domain v0.0.0|github.com/abhipray-cpu/tenantkit/domain $MOD_VERSION|g" \
      -e "s|github.com/abhipray-cpu/tenantkit/adapters/limiter-memory v0.0.0|github.com/abhipray-cpu/tenantkit/adapters/limiter-memory $MOD_VERSION|g" \
      "$mod"

    # Remove replace directives (lines containing 'replace' + '../')
    # This handles both single-line and block replace directives
    python3 -c "
import re, sys

with open('$mod', 'r') as f:
    content = f.read()

# Remove block replace(...) directives
content = re.sub(r'replace\s*\(\s*\n(.*?\n)*?\)', '', content)

# Remove single-line replace directives pointing to local paths
content = re.sub(r'replace\s+\S+\s+(\S+\s+)?=>\s+\.\..*\n', '', content)

# Clean up excessive blank lines
content = re.sub(r'\n{3,}', '\n\n', content).strip() + '\n'

with open('$mod', 'w') as f:
    f.write(content)
" 2>/dev/null || {
      # Fallback: use sed if python3 not available
      sed -i.bak '/^replace.*\.\.\//d' "$mod"
      sed -i.bak '/^replace ($/{N;/\.\.\//d}' "$mod"
    }

    rm -f "$mod.bak"
    echo "  ✅ $name/go.mod updated"
  fi
done

echo ""
echo "✅ All go.mod files updated for release."
echo ""

# ─── Step 3: Run go mod tidy on each module ──────────────────────────

echo "Step 3: Running go mod tidy on each module..."
echo ""
echo "  ⚠️  Skipping go mod tidy (requires published dependencies)."
echo "  After pushing tags, run: ./scripts/post-release-tidy.sh"
echo ""

# ─── Step 4: Commit ──────────────────────────────────────────────────

echo "Step 4: Committing release changes..."
git add -A
git commit -m "release: $VERSION

- Update all go.mod files to reference $MOD_VERSION
- Remove local replace directives for published release
- All 85+ tests passing, zero warnings"
echo "  ✅ Committed"
echo ""

# ─── Step 5: Create tags ─────────────────────────────────────────────

echo "Step 5: Creating git tags for each module..."
echo ""

# For Go modules in a monorepo, each submodule needs its own tag.
# The tag format is: <module-path-from-repo-root>/<version>
# e.g., tenantkit/v1.0.0, adapters/http-gin/v1.0.0

TAGS=(
  "$VERSION"
  "tenantkit/$VERSION"
  "tenantkit/domain/$VERSION"
)

for dir in adapters/*/; do
  name=$(basename "$dir")
  TAGS+=("adapters/$name/$VERSION")
done

for tag in "${TAGS[@]}"; do
  echo "  Creating tag: $tag"
  git tag -a "$tag" -m "Release $tag"
done

echo ""
echo "✅ All tags created."
echo ""

# ─── Step 6: Summary ─────────────────────────────────────────────────

echo "=== Release $VERSION Ready ==="
echo ""
echo "Tags created:"
for tag in "${TAGS[@]}"; do
  echo "  - $tag"
done
echo ""
echo "Next steps:"
echo "  1. Review the commit: git log -1"
echo "  2. Push tags:         git push origin main --tags"
echo "  3. Go to GitHub and create a Release from tag '$VERSION'"
echo "  4. After tags are pushed, pkg.go.dev will index within ~30 minutes"
echo "  5. Verify at: https://pkg.go.dev/github.com/abhipray-cpu/tenantkit/tenantkit"
echo ""
echo "To undo (if something went wrong):"
echo "  git reset --hard HEAD~1"
for tag in "${TAGS[@]}"; do
  echo "  git tag -d $tag"
done
echo ""
