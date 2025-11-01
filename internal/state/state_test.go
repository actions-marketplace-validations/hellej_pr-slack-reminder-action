package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hellej/pr-slack-reminder-action/internal/models"
)

func TestStateSaveAndLoadRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")

	originalState := State{
		SchemaVersion: CurrentSchemaVersion,
		CreatedAt:     time.Now().UTC(),
		SlackMessage: SlackRef{
			ChannelID: "C123456789",
			MessageTS: "1729123456.123456",
		},
		PullRequests: []PullRequestRef{
			{Repository: models.NewRepository("owner1", "repo1"), Number: 1},
			{Repository: models.NewRepository("owner1", "repo1"), Number: 2},
			{Repository: models.NewRepository("owner2", "repo2"), Number: 5},
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

	if loadedState.SlackMessage.ChannelID != originalState.SlackMessage.ChannelID {
		t.Errorf("SlackMessage.ChannelID mismatch: got %s, want %s", loadedState.SlackMessage.ChannelID, originalState.SlackMessage.ChannelID)
	}

	if loadedState.SlackMessage.MessageTS != originalState.SlackMessage.MessageTS {
		t.Errorf("SlackMessage.MessageTS mismatch: got %s, want %s", loadedState.SlackMessage.MessageTS, originalState.SlackMessage.MessageTS)
	}

	if len(loadedState.PullRequests) != len(originalState.PullRequests) {
		t.Errorf("PullRequests length mismatch: got %d, want %d", len(loadedState.PullRequests), len(originalState.PullRequests))
	}

	for i, pr := range loadedState.PullRequests {
		original := originalState.PullRequests[i]
		if pr.Repository.Owner != original.Repository.Owner || pr.Repository.Name != original.Repository.Name || pr.Number != original.Number {
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
		SlackMessage: SlackRef{
			ChannelID: "C123456789",
			MessageTS: "1729123456.123456",
		},
		PullRequests: []PullRequestRef{
			{Repository: models.NewRepository("owner1", "repo1"), Number: 1},
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
		SlackMessage: SlackRef{
			ChannelID: "C123456789",
			MessageTS: "1729123456.123456",
		},
		PullRequests: []PullRequestRef{
			{Repository: models.NewRepository("owner1", "repo1"), Number: 1},
		},
	}

	err := state.Validate()
	if err != nil {
		t.Errorf("Expected valid state to pass validation, got error: %v", err)
	}
}

func TestSaveSentSlackBlocksProperJSON(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "sent-blocks.json")

	slackBlocksJSON := []string{
		`{"type":"rich_text","block_id":"pr_list_heading","elements":[{"type":"rich_text_section","elements":[{"type":"text","text":"There are 2 open PRs 🚀","style":{"bold":true}}]}]}`,
		`{"type":"rich_text","block_id":"open_prs","elements":[{"type":"rich_text_list","elements":[{"type":"rich_text_section","elements":[{"type":"link","url":"https://github.com/owner/repo/pull/1","text":"Test PR","style":{"bold":true}}]}],"style":"bullet"}]}`,
	}

	err := SaveSentSlackBlocks(filePath, slackBlocksJSON)
	if err != nil {
		t.Fatalf("SaveSentSlackBlocks failed: %v", err)
	}

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	var savedBlocks []map[string]interface{}
	err = json.Unmarshal(fileContent, &savedBlocks)
	if err != nil {
		t.Fatalf("Saved file contains invalid JSON: %v", err)
	}

	if len(savedBlocks) != 2 {
		t.Errorf("Expected 2 blocks, got %d", len(savedBlocks))
	}

	// Verify the content doesn't contain escaped quotes (common sign of double-encoding)
	contentStr := string(fileContent)
	if strings.Contains(contentStr, `\"type\"`) {
		t.Error("Saved JSON contains escaped quotes, indicating double-encoding")
	}

	if len(savedBlocks) > 0 {
		if savedBlocks[0]["type"] != "rich_text" {
			t.Errorf("Expected first block type to be 'rich_text', got %v", savedBlocks[0]["type"])
		}
		if savedBlocks[0]["block_id"] != "pr_list_heading" {
			t.Errorf("Expected first block_id to be 'pr_list_heading', got %v", savedBlocks[0]["block_id"])
		}
	}
}

func TestSaveSentSlackBlocksEmptySlice(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "empty-blocks.json")

	err := SaveSentSlackBlocks(filePath, []string{})
	if err != nil {
		t.Fatalf("SaveSentSlackBlocks failed with empty slice: %v", err)
	}

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	var savedBlocks []map[string]interface{}
	err = json.Unmarshal(fileContent, &savedBlocks)
	if err != nil {
		t.Fatalf("Saved file contains invalid JSON: %v", err)
	}

	if len(savedBlocks) != 0 {
		t.Errorf("Expected empty array, got %d items", len(savedBlocks))
	}
}

func TestSaveSentSlackBlocksInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid-blocks.json")

	invalidJSON := []string{
		`{"type":"rich_text",`, // Incomplete JSON
	}

	err := SaveSentSlackBlocks(filePath, invalidJSON)
	if err == nil {
		t.Fatal("Expected error when saving invalid JSON, got nil")
	}

	if !strings.Contains(err.Error(), "failed to parse block") {
		t.Errorf("Expected JSON parse error, got: %v", err)
	}
}
