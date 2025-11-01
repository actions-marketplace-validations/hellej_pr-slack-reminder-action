package state

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hellej/pr-slack-reminder-action/internal/apiclients/slackclient"
	"github.com/hellej/pr-slack-reminder-action/internal/models"
	"github.com/hellej/pr-slack-reminder-action/internal/prparser"
	"github.com/hellej/pr-slack-reminder-action/internal/utilities"
)

const CurrentSchemaVersion = 1

type State struct {
	SchemaVersion int              `json:"schemaVersion"`
	CreatedAt     time.Time        `json:"createdAt"`
	SlackMessage  SlackRef         `json:"slackMessage"`
	PullRequests  []PullRequestRef `json:"pullRequests"`
}

type SlackRef struct {
	ChannelID string `json:"channelId"`
	MessageTS string `json:"messageTs"`
}

type PullRequestRef struct {
	Repository models.Repository `json:"repository"`
	Number     int               `json:"number"`
}

func PRToPullRequestRef(pr prparser.PR) PullRequestRef {
	return PullRequestRef{
		Repository: pr.Repository,
		Number:     *pr.Number,
	}
}

func Load(path string) (*State, error) {
	jsonData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state State
	if err := json.Unmarshal(jsonData, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func (s *State) Validate() error {
	if s.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("unsupported schema version %d, expected %d", s.SchemaVersion, CurrentSchemaVersion)
	}
	return nil
}

func SavePostState(
	filePath string,
	parsedPRs []prparser.PR,
	messageResponse *slackclient.MessageResponse,
) error {
	return savePostState(
		filePath,
		utilities.Map(parsedPRs, PRToPullRequestRef),
		SlackRef{
			ChannelID: messageResponse.ChannelID,
			MessageTS: messageResponse.Timestamp,
		})
}

func SaveSentSlackBlocks(
	filePath string,
	sentBlocks []string,
) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Parse JSON strings back to raw JSON objects to avoid double-encoding
	var parsedBlocks []json.RawMessage
	for i, blockJSON := range sentBlocks {
		var rawMessage json.RawMessage
		if err := json.Unmarshal([]byte(blockJSON), &rawMessage); err != nil {
			return fmt.Errorf("failed to parse block %d as JSON: %w", i, err)
		}
		parsedBlocks = append(parsedBlocks, rawMessage)
	}

	jsonData, err := json.MarshalIndent(parsedBlocks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sent blocks: %w", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write sent blocks file %s: %w", filePath, err)
	}

	return nil
}

func savePostState(filePath string, pullRequestRefs []PullRequestRef, slackRef SlackRef) error {
	stateToSave := State{
		SchemaVersion: CurrentSchemaVersion,
		CreatedAt:     time.Now(),
		SlackMessage:  slackRef,
		PullRequests:  pullRequestRefs,
	}

	if err := save(filePath, stateToSave); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}
	log.Printf("Saved state to %s with %d PRs", filePath, len(pullRequestRefs))
	return nil
}

func save(filePath string, state State) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	jsonData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write state file %s: %w", filePath, err)
	}

	return nil
}
