#!/usr/bin/env bash
set -euo pipefail

if ! command -v gh &> /dev/null; then
    echo "Error: GitHub CLI (gh) is not installed"
    echo "Install it from: https://cli.github.com/"
    exit 1
fi

if ! gh auth status &> /dev/null; then
    echo "Error: Not authenticated with GitHub CLI"
    echo "Run: gh auth login"
    exit 1
fi

REMOTE_URL=$(git remote get-url origin)
if [[ "$REMOTE_URL" =~ github\.com[:/]([^/]+)/([^/.]+)(\.git)?$ ]]; then
    REPO_OWNER="${BASH_REMATCH[1]}"
    REPO_NAME="${BASH_REMATCH[2]}"
    REPO="$REPO_OWNER/$REPO_NAME"
else
    echo "Error: Could not parse GitHub repository from remote URL: $REMOTE_URL"
    exit 1
fi

echo "Repository: $REPO"
echo ""

get_latest_tag() {
    git ls-remote --tags origin | awk '{print $2}' | grep -o 'refs/tags/v[0-9]*\.[0-9]*\.[0-9]*$' | sed 's_refs/tags/v__g' | sort -V | tail -n 1 | awk '{print "v"$1}'
}

LATEST_TAG=$(get_latest_tag)

if [[ -z "$LATEST_TAG" ]]; then
    echo "Error: No valid semver tags found"
    exit 1
fi

TAG_DATE=$(git log -1 --format=%ai "$LATEST_TAG" 2>/dev/null || echo "unknown")

echo "Latest release: $LATEST_TAG ($TAG_DATE)"
echo ""

COMMIT_COUNT=$(git rev-list --count "$LATEST_TAG..HEAD" 2>/dev/null || echo "0")

if [[ "$COMMIT_COUNT" -eq 0 ]]; then
    echo "⚠️  No commits to release since $LATEST_TAG"
    echo ""
    read -p "Proceed anyway? (y/n): " PROCEED
    if [[ "$PROCEED" != "y" ]]; then
        echo "Cancelled."
        exit 0
    fi
else
    echo "Commits since $LATEST_TAG ($COMMIT_COUNT commits):"
    echo ""
    git --no-pager log "$LATEST_TAG..HEAD" --format="%ai %h %s (%an)"
    echo ""
    read -p "Proceed with release? (y/n): " PROCEED
    if [[ "$PROCEED" != "y" ]]; then
        echo "Cancelled."
        exit 0
    fi
fi

echo ""
read -p "Select version bump (patch/minor/major) or 'exit' to cancel: " SEMVER

if [[ -z "$SEMVER" || "$SEMVER" == "exit" ]]; then
    echo "Cancelled."
    exit 0
fi

if [[ "$SEMVER" != "patch" && "$SEMVER" != "minor" && "$SEMVER" != "major" ]]; then
    echo "Error: Invalid semver option. Must be 'patch', 'minor', or 'major'"
    exit 1
fi

echo ""
echo "Triggering release workflow with semver=$SEMVER..."

gh workflow run release.yml -f semver="$SEMVER" --repo "$REPO"

if [[ $? -ne 0 ]]; then
    echo "Error: Failed to trigger workflow"
    exit 1
fi

echo "Waiting for workflow to start..."
sleep 2

RUN_URL=$(gh run list --workflow=release.yml --limit=1 --json url --jq '.[0].url' --repo "$REPO" 2>/dev/null || echo "")

echo ""
if [[ -n "$RUN_URL" ]]; then
    echo "✅ Release workflow triggered successfully!"
    echo "🚀 Workflow URL: $RUN_URL"
else
    echo "✅ Release workflow triggered successfully!"
    echo "🚀 View workflow runs at: https://github.com/$REPO/actions/workflows/release.yml"
fi
