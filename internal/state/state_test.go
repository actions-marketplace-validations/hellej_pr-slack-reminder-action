package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStateSaveAndLoadRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")

	originalState := State{
		SchemaVersion: CurrentSchemaVersion,
		CreatedAt:     time.Now().UTC(),
		Slack: SlackRef{
			ChannelID: "C123456789",
			MessageTS: "1729123456.123456",
		},
		PullRequests: []PullRequestRef{
			{Owner: "owner1", Repo: "repo1", Number: 1},
			{Owner: "owner1", Repo: "repo1", Number: 2},
			{Owner: "owner2", Repo: "repo2", Number: 5},
		},
	}

	err := save(statePath, originalState)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loadedState, err := Load(statePath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loadedState.SchemaVersion != originalState.SchemaVersion {
		t.Errorf("SchemaVersion mismatch: got %d, want %d", loadedState.SchemaVersion, originalState.SchemaVersion)
	}

	if !loadedState.CreatedAt.Equal(originalState.CreatedAt) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", loadedState.CreatedAt, originalState.CreatedAt)
	}

	if loadedState.Slack.ChannelID != originalState.Slack.ChannelID {
		t.Errorf("Slack.ChannelID mismatch: got %s, want %s", loadedState.Slack.ChannelID, originalState.Slack.ChannelID)
	}

	if loadedState.Slack.MessageTS != originalState.Slack.MessageTS {
		t.Errorf("Slack.MessageTS mismatch: got %s, want %s", loadedState.Slack.MessageTS, originalState.Slack.MessageTS)
	}

	if len(loadedState.PullRequests) != len(originalState.PullRequests) {
		t.Errorf("PullRequests length mismatch: got %d, want %d", len(loadedState.PullRequests), len(originalState.PullRequests))
	}

	for i, pr := range loadedState.PullRequests {
		original := originalState.PullRequests[i]
		if pr.Owner != original.Owner || pr.Repo != original.Repo || pr.Number != original.Number {
			t.Errorf("PullRequest[%d] mismatch: got %+v, want %+v", i, pr, original)
		}
	}
}

func TestLoadFileNotFound(t *testing.T) {
	nonExistentPath := "/tmp/non-existent-state.json"

	_, err := Load(nonExistentPath)
	if err == nil {
		t.Fatal("Expected error when loading non-existent file, got nil")
	}

	if !os.IsNotExist(err) {
		t.Errorf("Expected file not found error, got: %v", err)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	invalidJSONPath := filepath.Join(tempDir, "invalid.json")

	err := os.WriteFile(invalidJSONPath, []byte("{ invalid json content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid JSON file: %v", err)
	}

	_, err = Load(invalidJSONPath)
	if err == nil {
		t.Fatal("Expected error when loading invalid JSON, got nil")
	}

	var jsonErr *json.SyntaxError
	if !errors.As(err, &jsonErr) {
		t.Errorf("Expected JSON syntax error, got: %v", err)
	}
}

func TestStateValidateSchemaVersionMismatch(t *testing.T) {
	state := State{
		SchemaVersion: CurrentSchemaVersion + 1, // Wrong version
		CreatedAt:     time.Now().UTC(),
		Slack: SlackRef{
			ChannelID: "C123456789",
			MessageTS: "1729123456.123456",
		},
		PullRequests: []PullRequestRef{
			{Owner: "owner1", Repo: "repo1", Number: 1},
		},
	}

	err := state.Validate()
	if err == nil {
		t.Fatal("Expected validation error for schema version mismatch, got nil")
	}

	expectedMsg := "unsupported schema version"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain %q, got: %v", expectedMsg, err)
	}
}

func TestStateValidateValidState(t *testing.T) {
	state := State{
		SchemaVersion: CurrentSchemaVersion,
		CreatedAt:     time.Now().UTC(),
		Slack: SlackRef{
			ChannelID: "C123456789",
			MessageTS: "1729123456.123456",
		},
		PullRequests: []PullRequestRef{
			{Owner: "owner1", Repo: "repo1", Number: 1},
		},
	}

	err := state.Validate()
	if err != nil {
		t.Errorf("Expected valid state to pass validation, got error: %v", err)
	}
}
