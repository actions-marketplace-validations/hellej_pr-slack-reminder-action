# "Update" mode plan

Plan for adding support for a second run mode _update_ for the action.

## Run Modes
1. post (default): sends a PR reminder
2. update: updates that same reminder as PRs get reviewed/merged

Updated message contains exactly the same PR set as the original post (no new PRs included), regardless of whether they are now open, merged (or later optionally closed).

## High-Level Flow
post run:
- Fetch & filter PRs (current pipeline)
- Build + send Slack message
- Persist state (Slack identifiers + PR list)

update run:
- Load persisted state
- Re-fetch only those PRs in state (skip discovering new ones)
- Rebuild message (same deterministic PR ordering rules used in both modes)
- Mark merged PRs 🔀, refresh ages, reviewers, commenters
- Update existing Slack message in-place (chat.update)
- Add/update footer with refreshed timestamp (UTC)

Only post writes state. update is read-only regarding state (idempotent; does not alter file).

## State Contents
- Slack message reference (channel_id + message_ts)
- List of PR references (owner, repo, number)
- Version + created_at for evolution and staleness checks

### State JSON (MVP schema)
```json
{
  "schemaVersion": 1,
  "createdAt": "2025-10-15T12:34:56Z",
  "slack": {
    "channelId": "C123",
    "messageTs": "1728392.000200"
  },
  "pullRequests": [
    { "id": "owner1/repo1/1", "owner": "owner1", "repo": "repo1", "number": 1 },
    { "id": "owner1/repo1/2", "owner": "owner1", "repo": "repo1", "number": 2 }
  ]
}
```

Notes:
- pull_requests: `id` is a convenience key; structured fields to avoid reparsing.
- No reviewer/commenter snapshots in MVP; always re-derived on update.

## Storage Strategy
MVP: GitHub Cache action
- Pros: no extra permissions, quick to prototype
- Cons: eviction risk; not durable across long periods

Future: GitHub Actions Artifact
- Pros: controlled retention, explicit download, durable
- Needs: `actions: read` permission

Abstraction: read/write plain JSON file at configured path; the workflow decides how it is persisted (cache vs artifact). Avoid coupling core logic to the backend.

## Inputs (Proposed)
Minimal for MVP:
- `mode` (post|update) (default: post)
- `state-file-path` (default: `.pr-slack-reminder/state.json`)

Potential future inputs (defer unless needed):
- `state-cache-key` (override automatic key)
- `state-cache-ttl` (duration; warn/fail if state too old)
- `fail-if-missing-state` (bool, default true in update)
- `slack-update-retry-attempts` (default 3)
- `persistence` (cache|artifact)

## Message Update Behavior
- Recompute ages relative to current time
- Preserve original formatting (prefixes, sections)
- Add 🔀 indicator for merged PRs (suffix after reviewer info)
- Reviewer/commenter lists refreshed
- Footer replaced/added: `review info updated at HH:MM UTC`

## Edge Cases & Error Handling
| Scenario | Behavior (MVP) |
|----------|----------------|
| State file missing in update | Fail (clear error) |
| `fail-if-missing-state=false` | Log warning, exit success no-op |
| Slack message deleted | chat.update fails → error (future: optionally re-post) |
| PR merged | Show normally with 🔀 marker |
| PR closed (not merged) | Treat as still listed (no special marker MVP) |
| PR not found / permission lost | Skip with warning (future: placeholder) |
| State version != supported | Fail fast |
| State too old (if ttl configured) | Fail or warn depending on design (deferred) |
| Duplicate post with same path | Overwrite state silently (document) |
| Update retried | Safe; idempotent (same chat.update) |

## API Call Minimization
- Update mode fetches only PRs present in state (group PRs by repo).
- Single Slack update call per run.

## Idempotency
- Re-running post overwrites state and posts a brand-new message.
- Re-running update with unchanged PR states yields identical message payload.

## Testing Strategy (TDD)
Add failing tests before implementation:
1. Config parsing: default mode=post; invalid mode errors.
2. State save/load round trip (1 PR, multiple PRs).
3. Update: merged PR gains 🔀;
4. Missing state fails; optional no-op when flag false (when implemented).
5. Slack update error bubbled.
6. Version mismatch fails.

Integration test path (extend `cmd/pr-slack-reminder/main_test.go`):
- Simulate post → capture state file → mutate mock GitHub client to mark one PR merged → run update → assert mock Slack client received Update with expected blocks.

Mock enhancements:
- Extend mock Slack client with `UpdateMessage(channelID, ts, blocks)` recording.

## Implementation Steps (Recommended Order)
1. Add new input constant(s) + tests (mode parsing) → make pass.
2. Introduce `internal/state/` package (`State`, `PullRequestRef`, `Save`, `Load`, `Validate`).
3. Wire post path: after successful send, construct and save state.
4. Wire update path: load state, fetch targeted PRs, enrich, build message, call chat.update.
5. Add merged marker & footer injection (either via model enrichment or post-build pass).
6. Write/update tests for each step until green.
7. Refactor duplication (if any) after tests pass.

## Minimal Types (Draft)
State version constant: 

```go
// Decide on schema version form: keep int (simple comparisons) OR switch to semantic string.
// Current JSON example uses an integer (schemaVersion: 1). We'll keep int.
const CurrentSchemaVersion = 1

type SlackRef struct {
  ChannelID string `json:"channelId"`
  MessageTS string `json:"messageTs"`
}

type PullRequestRef struct {
  Owner  string `json:"owner"`
  Repo   string `json:"repo"`
  Number int    `json:"number"`
}

type State struct {
  SchemaVersion int              `json:"schemaVersion"`
  CreatedAt     time.Time        `json:"createdAt"`
  Slack         SlackRef         `json:"slack"`
  PullRequests  []PullRequestRef `json:"pullRequests"`
}

// Validation sketch:
// func (s *State) Validate() error { ... }
```

## Logging Guidelines
- post: `saved state path=... prs=N`
- update: `loaded state path=... prs=N age=...`
- warnings for skipped/missing PRs

## Risks & Mitigations
- Cache eviction → update failure: Document; later switch to artifact backend.
- Schema evolution: Version now; add migration path later if needed.
- Slack rate limit: Single update unlikely to hit; add simple retry/backoff.
- Permissions drift: Validate early, fail fast with actionable message.

## Future Enhancements (Not MVP)
- Artifact persistence backend
- Closed (not merged) indicator (🚫) vs merged (🔀)
- Multiple tracked messages (namespaced state keys)

## Auth
- Slack: chat.postMessage, chat.update only.
- GitHub: read PRs; later maybe actions:read for artifact strategy.
- No repository write permissions required.

---
This document defines the MVP scope and guardrails for implementing update mode with a clear TDD path and extensibility for future enhancements.
