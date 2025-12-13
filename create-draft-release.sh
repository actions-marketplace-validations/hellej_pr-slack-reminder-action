#!/usr/bin/env bash
set -euo pipefail

if ! command -v gh &> /dev/null; then
    echo "Error: GitHub CLI (gh) is not installed"
    echo "Install it from: https://cli.github.com/"
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

get_full_semver_tags() {
    git ls-remote --tags origin | \
        awk '{print $2}' | \
        grep -o 'refs/tags/v[0-9]*\.[0-9]*\.[0-9]*$' | \
        sed 's_refs/tags/__g' | \
        sort -V
}

ALL_TAGS=$(get_full_semver_tags)

if [[ -z "$ALL_TAGS" ]]; then
    echo "Error: No valid semver tags found"
    exit 1
fi

LATEST_TAG=$(echo "$ALL_TAGS" | tail -n 1)
echo "Latest tag: $LATEST_TAG"

PREVIOUS_TAG=$(echo "$ALL_TAGS" | tail -n 2 | head -n 1)

if [[ "$PREVIOUS_TAG" == "$LATEST_TAG" ]]; then
    echo "Error: Only one tag exists, cannot determine previous release"
    exit 1
fi

echo "Previous tag: $PREVIOUS_TAG"

COMMIT_COUNT=$(git rev-list --count "$PREVIOUS_TAG..$LATEST_TAG")
if [[ "$COMMIT_COUNT" -eq 0 ]]; then
    echo "Error: No commits between $PREVIOUS_TAG and $LATEST_TAG"
    exit 1
fi

echo "Found $COMMIT_COUNT commits between $PREVIOUS_TAG and $LATEST_TAG"

RELEASE_NOTES="## What's Changed"$'\n\n'

while IFS='|' read -r FULL_HASH AUTHOR_NAME COMMIT_MSG; do
    [[ -z "$FULL_HASH" ]] && continue
    
    SHORT_HASH="${FULL_HASH:0:7}"
    
    COMMIT_MSG_SINGLE_LINE=$(echo "$COMMIT_MSG" | tr '\n' ' ' | sed 's/  */ /g' | sed 's/^ *//;s/ *$//')
    
    if [[ "$AUTHOR_NAME" == *"[bot]"* ]]; then
        AUTHOR_ATTRIBUTION="$AUTHOR_NAME"
    else
        GH_USERNAME=$(gh api "repos/$REPO/commits/$FULL_HASH" --jq '.author.login' 2>/dev/null || echo "")
        
        if [[ -z "$GH_USERNAME" ]]; then
            AUTHOR_ATTRIBUTION="$AUTHOR_NAME"
        else
            AUTHOR_ATTRIBUTION="@$GH_USERNAME"
        fi
    fi
    
    COMMIT_URL="https://github.com/$REPO/commit/$FULL_HASH"
    COMMIT_LINE="- $COMMIT_MSG_SINGLE_LINE ([$SHORT_HASH]($COMMIT_URL)) $AUTHOR_ATTRIBUTION"
    
    RELEASE_NOTES="$RELEASE_NOTES$COMMIT_LINE"$'\n'
done < <(git log "$PREVIOUS_TAG..$LATEST_TAG" --format="%H|%an|%s")

RELEASE_NOTES="${RELEASE_NOTES%$'\n'}"

RELEASE_NOTES="$RELEASE_NOTES"$'\n\n'"**Full Changelog**: https://github.com/$REPO/compare/$PREVIOUS_TAG...$LATEST_TAG"

echo ""
echo "Creating draft release for $LATEST_TAG..."
RELEASE_OUTPUT=$(gh release create "$LATEST_TAG" \
    --draft \
    --title "$LATEST_TAG" \
    --notes "$RELEASE_NOTES" \
    --repo "$REPO" 2>&1)

if [[ $? -eq 0 ]]; then
    RELEASE_URL=$(echo "$RELEASE_OUTPUT" | grep -o 'https://github.com[^ ]*' | head -n 1)
    echo ""
    echo "✅ Draft release created successfully!"
    echo "📝 Release URL: $RELEASE_URL"
else
    echo "Error: Failed to create draft release"
    echo "$RELEASE_OUTPUT"
    exit 1
fi
